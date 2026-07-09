package hooks_test

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode/service/hooks"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddonHookPreDelete(t *testing.T) {
	// Setup real services backed by Postgres.
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	// Register the addon hook on the real taxcode service.
	addonHook, err := hooks.NewAddonHook(hooks.AddonHookConfig{AddonService: env.Addon})
	require.NoError(t, err)
	env.TaxCode.RegisterHooks(addonHook)

	ns := pctestutils.NewTestNamespace(t)

	// Provision organization-default tax codes so DeleteTaxCode can proceed past
	// the org-defaults check and reach the pre-delete hook.
	env.TaxCodeEnv.ProvisionDefaultTaxCodes(t, ns)

	t.Run("blocks deletion when an add-on references the tax code", func(t *testing.T) {
		// given: a tax code that an add-on will reference
		referenced, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "addon-referenced",
			Name:      "Referenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000003"},
			},
		})
		require.NoError(t, err)

		// given: an add-on whose rate card references the tax code
		addonInput := pctestutils.NewTestAddon(t, ns,
			&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "rc-1",
					Name: "RC 1",
					TaxConfig: &productcatalog.TaxConfig{
						TaxCodeID: lo.ToPtr(referenced.ID),
					},
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      decimal.NewFromInt(0),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
			},
		)
		addonInput.Key = "addon-with-taxcode"
		_, err = env.Addon.CreateAddon(t.Context(), addonInput)
		require.NoError(t, err)

		// when: attempting to delete the referenced tax code
		err = env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: referenced.ID},
		})

		// then: an error is returned and it is a TaxCodeReferencedByRateCard error
		require.Error(t, err)
		require.True(t, taxcode.IsTaxCodeReferencedByRateCardError(err),
			"expected TaxCodeReferencedByRateCard error, got: %v", err)
	})

	t.Run("allows deletion when no add-on references the tax code", func(t *testing.T) {
		// given: a tax code that no add-on references
		unreferenced, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "addon-unreferenced",
			Name:      "Unreferenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000004"},
			},
		})
		require.NoError(t, err)

		// when: deleting the unreferenced tax code
		err = env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: unreferenced.ID},
		})

		// then: no error is returned
		require.NoError(t, err)
	})
}
