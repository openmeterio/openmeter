package billingprofiles

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIBillingWorkflowSubscriptionEndProrationModeDefaults(t *testing.T) {
	createConfig, err := FromAPIBillingWorkflowCreate(api.BillingWorkflow{})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillActualPeriod, createConfig.Invoicing.SubscriptionEndProrationMode)

	updateConfig, err := FromAPIBillingWorkflow(api.BillingWorkflow{})
	require.NoError(t, err)
	require.Empty(t, updateConfig.Invoicing.SubscriptionEndProrationMode)

	explicitMode := api.BillingWorkflowInvoicingSubscriptionEndProrationModeBillFullPeriod
	explicitConfig, err := FromAPIBillingWorkflowCreate(api.BillingWorkflow{
		Invoicing: &api.BillingWorkflowInvoicingSettings{
			SubscriptionEndProrationMode: lo.ToPtr(explicitMode),
		},
	})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillFullPeriod, explicitConfig.Invoicing.SubscriptionEndProrationMode)
}

func TestToAPIBillingWorkflowDefaultTaxConfig(t *testing.T) {
	taxCode := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01JTAXCODE0000000000000000A"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
	}

	// given: a workflow whose default references a tax code mapped to txcd_10000000
	config := billing.DefaultWorkflowConfig
	config.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: taxCode.ID,
	}

	t.Run("deprecated provider codes come from the tax code's Stripe mapping", func(t *testing.T) {
		// when: it's rendered with the tax code resolved
		workflow, err := ToAPIBillingWorkflow(config, TaxCodes{taxCode.ID: taxCode})
		require.NoError(t, err)
		require.NotNil(t, workflow.Tax)

		// then: stripe and external_invoicing show the mapping, tax_code and tax_code_id the ID
		require.Equal(t, &api.BillingTaxConfig{
			Behavior:          lo.ToPtr(api.BillingTaxBehaviorInclusive),
			ExternalInvoicing: &api.BillingTaxConfigExternalInvoicing{Code: "txcd_10000000"},
			Stripe:            &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
			TaxCode:           &api.TaxCodeReference{Id: taxCode.ID},
			TaxCodeId:         lo.ToPtr(taxCode.ID),
		}, workflow.Tax.DefaultTaxConfig)
	})

	t.Run("without the tax code only the ID and behavior are shown", func(t *testing.T) {
		// when: it's rendered without the tax code
		workflow, err := ToAPIBillingWorkflow(config, nil)
		require.NoError(t, err)

		// then: no provider codes are shown
		require.Equal(t, &api.BillingTaxConfig{
			Behavior:  lo.ToPtr(api.BillingTaxBehaviorInclusive),
			TaxCode:   &api.TaxCodeReference{Id: taxCode.ID},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, workflow.Tax.DefaultTaxConfig)
	})

	t.Run("no default renders nothing", func(t *testing.T) {
		// when: a workflow without a default is rendered
		workflow, err := ToAPIBillingWorkflow(billing.DefaultWorkflowConfig, nil)
		require.NoError(t, err)

		// then: there's no default tax config
		require.Nil(t, workflow.Tax.DefaultTaxConfig)
	})
}

func TestFromAPIBillingTaxConfigFull(t *testing.T) {
	// given: a request with both tax_code.id and the deprecated tax_code_id
	// when: it's mapped
	raw, err := FromAPIBillingTaxConfigFull(&api.BillingTaxConfig{
		Behavior:  lo.ToPtr(api.BillingTaxBehaviorExclusive),
		Stripe:    &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
		TaxCode:   &api.TaxCodeReference{Id: "01JTAXCODE0000000000000000A"},
		TaxCodeId: lo.ToPtr("01JTAXCODE0000000000000000B"),
	})
	require.NoError(t, err)

	// then: tax_code.id wins and the deprecated stripe code is kept for the gate
	require.Equal(t, &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		TaxCodeID: lo.ToPtr("01JTAXCODE0000000000000000A"),
	}, raw)
}
