package billingprofiles

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
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

func TestBillingWorkflowEmptyInvoiceMode(t *testing.T) {
	unsetConfig, err := FromAPIBillingWorkflowCreate(api.BillingWorkflow{})
	require.NoError(t, err)
	require.Empty(t, unsetConfig.Invoicing.EmptyInvoiceMode, "the service resolves an unset mode, not the converter")

	explicitConfig, err := FromAPIBillingWorkflow(api.BillingWorkflow{
		Invoicing: &api.BillingWorkflowInvoicingSettings{
			EmptyInvoiceMode: lo.ToPtr(api.BillingWorkflowInvoicingEmptyInvoiceModeOmit),
		},
	})
	require.NoError(t, err)
	require.Equal(t, billing.EmptyInvoiceModeOmit, explicitConfig.Invoicing.EmptyInvoiceMode)

	workflow, err := ToAPIBillingWorkflow(explicitConfig)
	require.NoError(t, err)
	require.Equal(t, lo.ToPtr(api.BillingWorkflowInvoicingEmptyInvoiceModeOmit), workflow.Invoicing.EmptyInvoiceMode)
}
