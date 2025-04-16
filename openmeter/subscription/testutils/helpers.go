package subscriptiontestutils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CreateSubscriptionFromPlan(t *testing.T, deps *SubscriptionDependencies, planInp plan.CreatePlanInput, startAt time.Time) (subscription.Plan, subscription.SubscriptionView) {
	deps.FeatureConnector.CreateExampleFeatures(t)
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	plan := deps.PlanHelper.CreatePlan(t, planInp)
	subView, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  cust.Namespace,
		CustomerID: cust.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name: "test",
			Timing: subscription.Timing{
				Custom: &startAt,
			},
		},
	}, plan)
	require.NoError(t, err)

	return plan, subView
}

// For most cases, use the workflow service instead!
func CreateMultiInstanceAddonForSubscription(t *testing.T, deps *SubscriptionDependencies, subID models.NamespacedID, addonInp addon.CreateAddonInput, quants []subscriptionaddon.CreateSubscriptionAddonQuantityInput) (addon.Addon, subscriptionaddon.SubscriptionAddon) {
	t.Helper()

	add := deps.AddonService.CreateTestAddon(t, addonInp)

	subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
		AddonID:        add.ID,
		SubscriptionID: subID.ID,
		InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: quants[0].ActiveFrom,
			Quantity:   quants[0].Quantity,
		},
	})
	require.NoError(t, err)

	if len(quants) == 1 {
		return add, *subAdd
	}

	for _, quant := range quants[1:] {
		subAdd, err = deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, quant)
		require.NoError(t, err)
	}

	return add, *subAdd
}

// this is a bit hacky, we reuse the addon's effective period as cadence for the subscriptionaddon
// For most cases, use the workflow service instead!
func CreateAddonForSubscription(t *testing.T, deps *SubscriptionDependencies, subID models.NamespacedID, addonInp addon.CreateAddonInput) (addon.Addon, subscriptionaddon.SubscriptionAddon) {
	t.Helper()

	quants := []subscriptionaddon.CreateSubscriptionAddonQuantityInput{
		{
			ActiveFrom: *addonInp.EffectivePeriod.EffectiveFrom,
			Quantity:   1,
		},
	}

	if addonInp.EffectiveTo != nil {
		quants = append(quants, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: *addonInp.EffectiveTo,
			Quantity:   0,
		})
	}

	return CreateMultiInstanceAddonForSubscription(t, deps, subID, addonInp, quants)
}
