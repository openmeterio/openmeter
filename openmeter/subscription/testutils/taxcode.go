package subscriptiontestutils

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
)

const (
	ExampleTaxCodeKey = "example"
	// ExampleStripeTaxCode is the Stripe code the example rate cards carried before they referenced
	// tax codes by ID, so the legacy Stripe path resolves to the same tax code.
	ExampleStripeTaxCode = "txcd_10000000"
)

// ExampleTaxConfig returns the tax config of the example rate cards. Every call in a namespace
// shares one tax code: plan versions built from separate calls must compare equal, otherwise
// migrations would split every taxed item.
func ExampleTaxConfig(t *testing.T, env *taxcodetestutils.TestEnv) productcatalog.TaxConfig {
	t.Helper()
	require.NotNil(t, env, "the example tax code needs a tax code env; pass WithTaxConfig to build without one")

	tc, err := env.Service.GetTaxCodeByKey(t.Context(), taxcode.GetTaxCodeByKeyInput{
		Namespace: ExampleNamespace,
		Key:       ExampleTaxCodeKey,
	})
	if err != nil {
		require.True(t, taxcode.IsTaxCodeNotFoundError(err), "getting example tax code: %v", err)

		tc = env.CreateTaxCode(t, ExampleNamespace, taxcode.CreateTaxCodeInput{
			Key:  ExampleTaxCodeKey,
			Name: "Example",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: ExampleStripeTaxCode},
			},
		})
	}

	return productcatalog.TaxConfig{
		TaxCodeID: lo.ToPtr(tc.ID),
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}
}
