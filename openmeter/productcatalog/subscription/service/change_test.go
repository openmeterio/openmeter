package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestChange(t *testing.T) {
	logger := testutils.NewLogger(t)

	examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

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

	t.Run("Should change to different plan", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

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
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t)

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
							Custom: lo.ToPtr(now.Add(time.Second)),
						},
					},
					Namespace:  cust.Namespace,
					CustomerID: cust.ID,
				},
			})
			require.Nil(t, err)

			p2Input := examplePlanInput1
			p2Input.Plan.PlanMeta.Name = "New Plan"
			p2Input.Plan.PlanMeta.Key = "new_plan"
			// We need to copy the phases to avoid modifying the original plan
			p2Input.Plan.Phases = lo.Map(p2Input.Plan.Phases, func(phase productcatalog.Phase, _ int) productcatalog.Phase { return phase })
			p2Input.Plan.Phases[2].Duration = lo.ToPtr(testutils.GetISODuration(t, "P5M"))
			p2Input.Plan.Phases = append(p2Input.Plan.Phases, productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "test_phase_4",
					Name:        "Test Phase 4",
					Description: lo.ToPtr("Test Phase 4 Description"),
				},
				RateCards: productcatalog.RateCards{
					&subscriptiontestutils.ExampleRateCard1,
				},
			})

			// Let's create a second plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, p2Input)
			require.Nil(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new plan
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.Nil(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			pInp := plansubscription.PlanInput{}
			pInp.FromRef(&plansubscription.PlanRefInput{
				Key:     plan2.Key,
				Version: &plan2.Version,
			})

			// Let's change the subscription to the new plan
			resp, err := svc.Change(ctx, plansubscription.ChangeSubscriptionRequest{
				ID: sub.NamespacedID,
				WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: lo.ToPtr(clock.Now()),
					},
					Name: sub.Name,
				},
				PlanInput: pInp,
			})
			require.Nil(t, err)

			require.Equal(t, sub.NamespacedID, resp.Current.NamespacedID)
			require.Equal(t, plan2.PlanMeta.Key, resp.Next.Subscription.PlanRef.Key)
		})
	})

	t.Run("Should not allow changing to inactive plan", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

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
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t)

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
							Custom: lo.ToPtr(now.Add(time.Second)),
						},
					},
					Namespace:  cust.Namespace,
					CustomerID: cust.ID,
				},
			})
			require.Nil(t, err)

			p2Input := examplePlanInput1
			p2Input.Plan.PlanMeta.Name = "New Plan"
			p2Input.Plan.PlanMeta.Key = "new_plan"
			// We need to copy the phases to avoid modifying the original plan
			p2Input.Plan.Phases = lo.Map(p2Input.Plan.Phases, func(phase productcatalog.Phase, _ int) productcatalog.Phase { return phase })
			p2Input.Plan.Phases[2].Duration = lo.ToPtr(testutils.GetISODuration(t, "P5M"))
			p2Input.Plan.Phases = append(p2Input.Plan.Phases, productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "test_phase_4",
					Name:        "Test Phase 4",
					Description: lo.ToPtr("Test Phase 4 Description"),
				},
				RateCards: productcatalog.RateCards{
					&subscriptiontestutils.ExampleRateCard1,
				},
			})

			// Let's create a second plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, p2Input)
			require.Nil(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new plan
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.Nil(t, err)
			require.NotNil(t, plan2)

			// Let's create a new version of the second plan
			p2v2Input := p2Input
			p2v2Input.Plan.PlanMeta.Name = "New Plan 2"

			plan2v2, err := deps.subDeps.PlanService.CreatePlan(ctx, p2v2Input)
			require.Nil(t, err)

			eFrom2 := clock.Now().Add(10 * time.Second)

			// Let's publish the new version of the second plan
			_, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2v2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom2,
				},
			})
			require.Nil(t, err)

			clock.SetTime(eFrom2.Add(time.Second))

			// And let's try to change to the old plan still
			pInp := plansubscription.PlanInput{}
			pInp.FromRef(&plansubscription.PlanRefInput{
				Key:     plan2.Key,
				Version: &plan2.Version,
			})

			// Let's change the subscription to the new plan
			_, err = svc.Change(ctx, plansubscription.ChangeSubscriptionRequest{
				ID: sub.NamespacedID,
				WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: lo.ToPtr(clock.Now()),
					},
					Name: sub.Name,
				},
				PlanInput: pInp,
			})

			require.NotNil(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
			require.ErrorContains(t, err, "plan is not active")
		})
	})
}
