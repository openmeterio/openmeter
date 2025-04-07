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

func TestMigrate(t *testing.T) {
	logger := testutils.NewLogger(t)

	examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

	type tDeps struct {
		subDeps subscriptiontestutils.SubscriptionDependencies
		subSvc  subscription.Service
		wfSvc   subscriptionworkflow.Service
	}

	withDeps := func(t *testing.T, f func(t *testing.T, deps tDeps)) {
		t.Helper()
		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)

		f(t, tDeps{
			subDeps: deps,
			subSvc:  deps.SubscriptionService,
			wfSvc:   deps.WorkflowService,
		})
	}

	t.Run("Should migrate to latest version of plan when none is specified", func(t *testing.T) {
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

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.Nil(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.Nil(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			// Let's migrate the subscription to the new version
			resp, err := svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID:            sub.NamespacedID,
				TargetVersion: &plan2.PlanMeta.Version,
			})
			require.Nil(t, err)

			require.Equal(t, sub.NamespacedID, resp.Current.NamespacedID)
			require.Equal(t, plan2.PlanMeta.Version, resp.Next.Subscription.PlanRef.Version)
		})
	})

	t.Run("Should not allow migrating to same or smaller version", func(t *testing.T) {
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

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.Nil(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.Nil(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			// Let's migrate the subscription to the new version
			_, err = svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID:            sub.NamespacedID,
				TargetVersion: lo.ToPtr(plan1.ToCreateSubscriptionPlanInput().Plan.Version),
			})
			require.NotNil(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should not allow migrating to archived version", func(t *testing.T) {
		t.Skip("Should it or should it not? Right now it allows it")
	})
}
