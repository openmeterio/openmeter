package taxcoderesolver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/taxcoderesolver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestTaxCodeResolverFactory(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)

	factory, err := taxcoderesolver.NewFactory(env.Service, env.Adapter)
	require.NoError(t, err)

	resolve := func(t *testing.T, input productcatalog.ResolveOrBackfillTaxCodeInput) (*taxcode.TaxCode, error) {
		t.Helper()

		var tc *taxcode.TaxCode

		err := factory.WithResolver(t.Context(), func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
			var err error
			tc, err = resolver.ResolveOrBackfillTaxCode(ctx, input)

			return err
		})

		return tc, err
	}

	getByStripeCode := func(t *testing.T, namespace, stripeCode string) (taxcode.TaxCode, error) {
		t.Helper()

		return env.Service.GetTaxCodeByAppMapping(t.Context(), taxcode.GetTaxCodeByAppMappingInput{
			Namespace: namespace,
			AppType:   app.AppTypeStripe,
			TaxCode:   stripeCode,
		})
	}

	t.Run("ExistingMappingReturnsExistingTaxCode", func(t *testing.T) {
		// given: a tax code mapped to the stripe code
		namespace := testutils.NameGenerator.Generate().Key
		existing := env.CreateTaxCode(t, namespace, taxcode.CreateTaxCodeInput{
			AppMappings: taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
		})

		// when: resolving the stripe code
		got, err := resolve(t, productcatalog.ResolveOrBackfillTaxCodeInput{
			Namespace:  namespace,
			StripeCode: "txcd_10000000",
		})

		// then: the existing tax code is returned
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, existing.ID, got.ID)
	})

	t.Run("UnknownStripeCodeCreatesTaxCode", func(t *testing.T) {
		// given: no tax code mapped to the stripe code
		namespace := testutils.NameGenerator.Generate().Key

		// when: resolving the stripe code
		got, err := resolve(t, productcatalog.ResolveOrBackfillTaxCodeInput{
			Namespace:  namespace,
			StripeCode: "txcd_20000000",
		})

		// then: a tax code mapped to it is created and committed
		require.NoError(t, err)

		created, err := getByStripeCode(t, namespace, "txcd_20000000")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, created.ID, got.ID)
	})

	t.Run("FactoryReadsInTheCallbackSeeCreatedTaxCode", func(t *testing.T) {
		// given: no tax code mapped to the stripe code
		namespace := testutils.NameGenerator.Generate().Key

		// when: the callback creates one and then batch-resolves it through the factory with the
		// callback's ctx
		err := factory.WithResolver(t.Context(), func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
			created, err := resolver.ResolveOrBackfillTaxCode(ctx, productcatalog.ResolveOrBackfillTaxCodeInput{
				Namespace:  namespace,
				StripeCode: "txcd_40000000",
			})
			if err != nil {
				return err
			}

			got, err := factory.WithNamespace(namespace).BatchResolve(ctx, created.ID)
			if err != nil {
				return err
			}

			// then: the uncommitted tax code is visible inside the transaction
			assert.Contains(t, got, created.ID)

			return nil
		})
		require.NoError(t, err)
	})

	t.Run("CallbackErrorRollsBackCreatedTaxCode", func(t *testing.T) {
		// given: no tax code mapped to the stripe code
		namespace := testutils.NameGenerator.Generate().Key
		domainErr := errors.New("domain operation failed")

		var otherKey string

		// when: the callback creates a tax code through the resolver and writes something
		// else, then fails
		err := factory.WithResolver(t.Context(), func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
			if _, err := resolver.ResolveOrBackfillTaxCode(ctx, productcatalog.ResolveOrBackfillTaxCodeInput{
				Namespace:  namespace,
				StripeCode: "txcd_30000000",
			}); err != nil {
				return err
			}

			other, err := env.Service.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
				Namespace: namespace,
				Key:       "domain_write",
				Name:      "Domain write",
			})
			if err != nil {
				return err
			}

			otherKey = other.Key

			return domainErr
		})

		// then: the callback's error is returned and neither write is kept
		require.ErrorIs(t, err, domainErr)
		require.NotEmpty(t, otherKey)

		_, err = getByStripeCode(t, namespace, "txcd_30000000")
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)

		_, err = env.Service.GetTaxCodeByKey(t.Context(), taxcode.GetTaxCodeByKeyInput{
			Namespace: namespace,
			Key:       otherKey,
		})
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)
	})

	t.Run("InvalidInput", func(t *testing.T) {
		for name, input := range map[string]productcatalog.ResolveOrBackfillTaxCodeInput{
			"missing namespace":   {StripeCode: "txcd_10000000"},
			"missing stripe code": {Namespace: testutils.NameGenerator.Generate().Key},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := resolve(t, input)

				require.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
			})
		}
	})
}

func TestNewFactoryRequiresDependencies(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)

	_, err := taxcoderesolver.NewFactory(nil, env.Adapter)
	require.Error(t, err)

	_, err = taxcoderesolver.NewFactory(env.Service, nil)
	require.Error(t, err)
}
