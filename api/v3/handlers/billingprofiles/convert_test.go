package billingprofiles

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
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

	workflow, err := ToAPIBillingWorkflow(explicitConfig)
	require.NoError(t, err)
	require.Equal(t, lo.ToPtr(apiv3models.BillingWorkflowInvoicingEmptyInvoiceModeOmit), workflow.Invoicing.EmptyInvoiceMode)
}
