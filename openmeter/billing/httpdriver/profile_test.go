package httpdriver

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestFromAPIBillingWorkflowSubscriptionEndProrationModeDefaults(t *testing.T) {
	createConfig, err := fromAPIBillingWorkflowCreate(apiv1models.BillingWorkflowCreate{})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillActualPeriod, createConfig.Invoicing.SubscriptionEndProrationMode)

	updateConfig, err := fromAPIBillingWorkflow(apiv1models.BillingWorkflow{})
	require.NoError(t, err)
	require.Empty(t, updateConfig.Invoicing.SubscriptionEndProrationMode)

	explicitMode := apiv1models.BillingWorkflowInvoicingSubscriptionEndProrationModeBillFullPeriod
	explicitConfig, err := fromAPIBillingWorkflowCreate(apiv1models.BillingWorkflowCreate{
		Invoicing: &apiv1models.BillingWorkflowInvoicingSettings{
			SubscriptionEndProrationMode: lo.ToPtr(explicitMode),
		},
	})
	require.NoError(t, err)
	require.Equal(t, billing.SubscriptionEndProrationModeBillFullPeriod, explicitConfig.Invoicing.SubscriptionEndProrationMode)
}
