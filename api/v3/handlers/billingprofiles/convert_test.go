package billingprofiles

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIBillingWorkflowSubscriptionEndProrationModeDefaults(t *testing.T) {
	createConfig, err := FromAPIBillingWorkflowCreate(apiv3models.BillingWorkflow{})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillActualPeriod, createConfig.Invoicing.SubscriptionEndProrationMode)

	updateConfig, err := FromAPIBillingWorkflow(apiv3models.BillingWorkflow{})
	require.NoError(t, err)
	require.Empty(t, updateConfig.Invoicing.SubscriptionEndProrationMode)

	explicitMode := apiv3models.BillingWorkflowInvoicingSubscriptionEndProrationModeBillFullPeriod
	explicitConfig, err := FromAPIBillingWorkflowCreate(apiv3models.BillingWorkflow{
		Invoicing: &apiv3models.BillingWorkflowInvoicingSettings{
			SubscriptionEndProrationMode: lo.ToPtr(explicitMode),
		},
	})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillFullPeriod, explicitConfig.Invoicing.SubscriptionEndProrationMode)
}

func TestBillingWorkflowEmptyInvoiceMode(t *testing.T) {
	unsetConfig, err := FromAPIBillingWorkflowCreate(apiv3models.BillingWorkflow{})
	require.NoError(t, err)
	require.Empty(t, unsetConfig.Invoicing.EmptyInvoiceMode, "the service resolves an unset mode, not the converter")

	explicitConfig, err := FromAPIBillingWorkflow(apiv3models.BillingWorkflow{
		Invoicing: &apiv3models.BillingWorkflowInvoicingSettings{
			EmptyInvoiceMode: lo.ToPtr(apiv3models.BillingWorkflowInvoicingEmptyInvoiceModeOmit),
		},
	})
	require.NoError(t, err)
	require.Equal(t, billing.EmptyInvoiceModeOmit, explicitConfig.Invoicing.EmptyInvoiceMode)

	workflow, err := ToAPIBillingWorkflow(explicitConfig, nil)
	require.NoError(t, err)
	require.Equal(t, lo.ToPtr(apiv3models.BillingWorkflowInvoicingEmptyInvoiceModeOmit), workflow.Invoicing.EmptyInvoiceMode)
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
		require.Equal(t, &apiv3models.BillingTaxConfig{
			Behavior:          lo.ToPtr(apiv3models.BillingTaxBehaviorInclusive),
			ExternalInvoicing: &apiv3models.BillingTaxConfigExternalInvoicing{Code: "txcd_10000000"},
			Stripe:            &apiv3models.BillingTaxConfigStripe{Code: "txcd_10000000"},
			TaxCode:           &apiv3models.TaxCodeReference{Id: taxCode.ID},
			TaxCodeId:         lo.ToPtr(taxCode.ID),
		}, workflow.Tax.DefaultTaxConfig)
	})

	t.Run("without the tax code only the ID and behavior are shown", func(t *testing.T) {
		// when: it's rendered without the tax code
		workflow, err := ToAPIBillingWorkflow(config, nil)
		require.NoError(t, err)

		// then: no provider codes are shown
		require.Equal(t, &apiv3models.BillingTaxConfig{
			Behavior:  lo.ToPtr(apiv3models.BillingTaxBehaviorInclusive),
			TaxCode:   &apiv3models.TaxCodeReference{Id: taxCode.ID},
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
	raw, err := FromAPIBillingTaxConfigFull(&apiv3models.BillingTaxConfig{
		Behavior:  lo.ToPtr(apiv3models.BillingTaxBehaviorExclusive),
		Stripe:    &apiv3models.BillingTaxConfigStripe{Code: "txcd_10000000"},
		TaxCode:   &apiv3models.TaxCodeReference{Id: "01JTAXCODE0000000000000000A"},
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
