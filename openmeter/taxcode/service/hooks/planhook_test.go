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

// setupNamespaceDefaults provisions the organisation-default tax codes that
// DeleteTaxCode requires to exist before it calls the pre-delete hook.
func setupNamespaceDefaults(t *testing.T, env *pctestutils.TestEnv, ns string) {
	t.Helper()

	invoicing, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "default-invoicing",
		Name:      "Provider Default",
	})
	require.NoError(t, err)

	creditGrant, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "default-credit-grant",
		Name:      "Non-Taxable",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_00000000"},
		},
	})
	require.NoError(t, err)

	_, err = env.TaxCode.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
		Namespace:            ns,
		InvoicingTaxCodeID:   invoicing.ID,
		CreditGrantTaxCodeID: creditGrant.ID,
	})
	require.NoError(t, err)
}

func TestPlanHookPreDelete(t *testing.T) {
	// Setup real services backed by Postgres.
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	// Register the plan hook on the real taxcode service.
	planHook, err := hooks.NewPlanHook(hooks.PlanHookConfig{PlanService: env.Plan})
	require.NoError(t, err)
	env.TaxCode.RegisterHooks(planHook)

	ns := pctestutils.NewTestNamespace(t)

	// Provision organisation-default tax codes so DeleteTaxCode can proceed past
	// the org-defaults check and reach the pre-delete hook.
	setupNamespaceDefaults(t, env, ns)

	t.Run("blocks deletion when a plan references the tax code", func(t *testing.T) {
		// given: a tax code that a plan will reference
		referenced, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "referenced",
			Name:      "Referenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000001"},
			},
		})
		require.NoError(t, err)

		// given: a plan whose rate card references the tax code
		planInput := pctestutils.NewTestPlan(t, ns,
			pctestutils.WithPlanKey("plan-with-taxcode"),
			pctestutils.WithPlanPhases(productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:  "default",
					Name: "Default",
				},
				RateCards: []productcatalog.RateCard{
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
						BillingCadence: &pctestutils.MonthPeriod,
					},
				},
			}),
		)
		_, err = env.Plan.CreatePlan(t.Context(), planInput)
		require.NoError(t, err)

		// when: attempting to delete the referenced tax code
		err = env.TaxCode.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: referenced.ID},
		})

		// then: an error is returned and it is a TaxCodeReferencedByPlan error
		require.Error(t, err)
		require.True(t, taxcode.IsTaxCodeReferencedByPlanError(err),
			"expected TaxCodeReferencedByPlan error, got: %v", err)
	})

	t.Run("allows deletion when no plan references the tax code", func(t *testing.T) {
		// given: a tax code that no plan references
		unreferenced, err := env.TaxCode.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "unreferenced",
			Name:      "Unreferenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000002"},
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
