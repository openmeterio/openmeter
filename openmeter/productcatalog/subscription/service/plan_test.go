package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestDiscountPersisting(t *testing.T) {
	logger := testutils.NewLogger(t)

	type tDeps struct {
		subDeps subscriptiontestutils.ExposedServiceDeps
		subSvc  subscription.Service
		wfSvc   subscriptionworkflow.Service
	}

	withDeps := func(t *testing.T, f func(t *testing.T, deps tDeps)) {
		t.Helper()
		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		svc, exposedDeps := subscriptiontestutils.NewService(t, dbDeps)

		f(t, tDeps{
			subDeps: exposedDeps,
			subSvc:  svc.Service,
			wfSvc:   svc.WorkflowService,
		})
	}

	t.Run("Should persist discounts", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)
			examplePlanInput1.Phases[0].RateCards[0] = &subscriptiontestutils.ExampleRateCardWithDiscounts

			ctx := context.Background()

			svc := service.New(service.Config{
				SubscriptionService: deps.subSvc,
				WorkflowService:     deps.wfSvc,
				Logger:              logger,
				PlanService:         deps.subDeps.PlanService,
				CustomerService:     deps.subDeps.CustomerService,
			})

			// Let's set up the feature & customer
			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeature(t)

			// Let's create the plan
			plan1 := deps.subDeps.PlanHelper.CreatePlan(t, examplePlanInput1)

			// Let's create the subscription
			p1Inp := plansubscription.PlanInput{}
			p1Inp.FromRef(&plansubscription.PlanRefInput{
				Key:     plan1.ToCreateSubscriptionPlanInput().Plan.Key,
				Version: &plan1.ToCreateSubscriptionPlanInput().Plan.Version,
			})

			sub, err := svc.Create(ctx, plansubscription.CreateSubscriptionRequest{
				PlanInput: p1Inp,
				WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
						Name: "test",
						Timing: subscription.Timing{
							Enum: lo.ToPtr(subscription.TimingImmediate),
						},
					},
					Namespace:  cust.Namespace,
					CustomerID: cust.ID,
				},
			})
			require.Nil(t, err)

			subView, err := deps.subSvc.GetView(ctx, sub.NamespacedID)
			require.Nil(t, err)

			require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey], 1)
			item := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0]

			require.Len(t, item.Spec.RateCard.Discounts, 1)

			discount, err := item.Spec.RateCard.Discounts[0].AsPercentage()
			require.NoError(t, err)
			require.Equal(t, "10%", discount.Percentage.String())
		})
	})
}
