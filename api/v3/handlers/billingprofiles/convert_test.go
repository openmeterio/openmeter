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
