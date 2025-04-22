package subscriptiontestutils

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func CreatePlanWithAddon(
	t *testing.T,
	deps SubscriptionDependencies,
	planInp plan.CreatePlanInput,
	addonInp addon.CreateAddonInput,
) (subscription.Plan, addon.Addon) {
	t.Helper()

	_ = deps.FeatureConnector.CreateExampleFeatures(t)

	p, err := deps.PlanService.CreatePlan(context.Background(), planInp)
	require.Nil(t, err)
	require.NotNil(t, p)

	add := deps.AddonService.CreateTestAddon(t, addonInp)

	_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ExampleNamespace,
		},
		PlanID:        p.ID,
		AddonID:       add.ID,
		FromPlanPhase: p.Phases[0].Key,
	})
	require.Nil(t, err, "received error: %s", err)

	p, err = deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now()),
			EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
		},
	})
	require.Nil(t, err, "received error: %s", err)

	return &plansubscription.Plan{
		Plan: p.AsProductCatalogPlan(),
		Ref:  &p.NamespacedID,
	}, add
}

func CreateSubscriptionFromPlan(t *testing.T, deps *SubscriptionDependencies, plan subscription.Plan, startAt time.Time) subscription.SubscriptionView {
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

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

	return subView
}

// For most cases, use the workflow service instead!
func CreateMultiInstanceAddonForSubscription(t *testing.T, deps *SubscriptionDependencies, subID models.NamespacedID, addonID models.NamespacedID, quants []subscriptionaddon.CreateSubscriptionAddonQuantityInput) subscriptionaddon.SubscriptionAddon {
	t.Helper()

	subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
		AddonID:        addonID.ID,
		SubscriptionID: subID.ID,
		InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: quants[0].ActiveFrom,
			Quantity:   quants[0].Quantity,
		},
	})
	require.NoError(t, err)

	if len(quants) == 1 {
		return *subAdd
	}

	for _, quant := range quants[1:] {
		subAdd, err = deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, quant)
		require.NoError(t, err)
	}

	return *subAdd
}

func CreateAddonForSubscription(t *testing.T, deps *SubscriptionDependencies, subID models.NamespacedID, addonID models.NamespacedID, cadence models.CadencedModel) subscriptionaddon.SubscriptionAddon {
	t.Helper()

	quants := []subscriptionaddon.CreateSubscriptionAddonQuantityInput{
		{
			ActiveFrom: cadence.ActiveFrom,
			Quantity:   1,
		},
	}

	if cadence.ActiveTo != nil {
		quants = append(quants, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: *cadence.ActiveTo,
			Quantity:   0,
		})
	}

	return CreateMultiInstanceAddonForSubscription(t, deps, subID, addonID, quants)
}
