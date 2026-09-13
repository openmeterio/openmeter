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
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMigrate(t *testing.T) {
	logger := testutils.NewLogger(t)

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
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			// Let's set up the feature & customer
			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

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
			require.NoError(t, err)

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.NoError(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			// Let's migrate the subscription to the new version
			resp, err := svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID: sub.NamespacedID,
				Timing: &subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				},
			})
			require.NoError(t, err)

			require.Equal(t, sub.NamespacedID, resp.Current.NamespacedID)
			require.Equal(t, plan2.PlanMeta.Version, resp.Next.Subscription.PlanRef.Version)
		})
	})

	t.Run("Should migrate to specific version of plan when provided", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			// Let's set up the feature & customer
			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

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
			require.NoError(t, err)

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.NoError(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			// Let's migrate the subscription to the new version
			resp, err := svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID:            sub.NamespacedID,
				TargetVersion: &plan2.PlanMeta.Version,
			})
			require.NoError(t, err)

			require.Equal(t, sub.NamespacedID, resp.Current.NamespacedID)
			require.Equal(t, plan2.PlanMeta.Version, resp.Next.Subscription.PlanRef.Version)
		})
	})

	t.Run("Should not allow migrating to same or smaller version", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			// Let's set up the feature & customer
			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

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
			require.NoError(t, err)

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.NoError(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			// Let's migrate the subscription to the new version
			_, err = svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID:            sub.NamespacedID,
				TargetVersion: lo.ToPtr(plan1.ToCreateSubscriptionPlanInput().Plan.Version),
				Timing: &subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				},
			})
			require.NotNil(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should not allow migrating to archived version", func(t *testing.T) {
		t.Skip("Should it or should it not? Right now it allows it")
	})

	t.Run("Should reject phase resets during migration", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			// Let's set up the feature & customer
			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

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
			require.NoError(t, err)

			pv2Input := examplePlanInput1
			pv2Input.Plan.PlanMeta.Name = "New Name"

			// Let's create a new version of the plan
			plan2, err := deps.subDeps.PlanService.CreatePlan(ctx, pv2Input)
			require.NoError(t, err)

			eFrom := clock.Now().Add(5 * time.Second)

			// Let's publish the new version
			plan2, err = deps.subDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID: plan2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: &eFrom,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, plan2)

			clock.SetTime(eFrom.Add(time.Second))

			t.Run("Should error if starting phase is not found", func(t *testing.T) {
				// Let's migrate the subscription to the new version starting with the second phase
				_, err := svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
					ID:            sub.NamespacedID,
					TargetVersion: &plan2.PlanMeta.Version,
					StartingPhase: lo.ToPtr("test_phase_NOT_FOUND"),
					Timing: &subscription.Timing{
						Enum: lo.ToPtr(subscription.TimingImmediate),
					},
				})
				require.Error(t, err)
				require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
			})

			_, err = svc.Migrate(ctx, plansubscription.MigrateSubscriptionRequest{
				ID:            sub.NamespacedID,
				TargetVersion: &plan2.PlanMeta.Version,
				StartingPhase: lo.ToPtr("test_phase_2"),
			})
			require.ErrorContains(t, err, "preserves the phase timeline")
		})
	})
}

// Commit another migration after the catalog service reads the subscription,
// before its migration workflow acquires the customer lock.
type migrateAfterSubscriptionRead struct {
	subscription.Service
	workflow subscriptionworkflow.Service
	plan     subscription.Plan
	migrated subscription.SubscriptionView
}

func (s *migrateAfterSubscriptionRead) Get(ctx context.Context, id models.NamespacedID) (subscription.Subscription, error) {
	before, err := s.Service.Get(ctx, id)
	if err != nil {
		return subscription.Subscription{}, err
	}

	_, s.migrated, err = s.workflow.MigrateToPlan(ctx, subscriptionworkflow.MigrateSubscriptionWorkflowInput{
		SubscriptionID: id,
		Plan:           s.plan,
		Timing:         subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
	})

	return before, err
}

func TestMigrateReturnsCurrentSnapshotFromUnderLock(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a subscription on version 1 and two later plan versions
	db := subscriptiontestutils.SetupDBDeps(t)
	defer db.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, db)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
		subscriptiontestutils.ExampleRateCard1.Clone(),
	).Build()
	p1 := deps.PlanHelper.CreatePlan(t, planInput)
	before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p1, start)

	clock.FreezeTime(start.Add(24 * time.Hour))
	defer clock.UnFreeze()
	p2 := deps.PlanHelper.CreatePlan(t, planInput)

	clock.FreezeTime(start.Add(48 * time.Hour))
	defer clock.UnFreeze()
	p3 := deps.PlanHelper.CreatePlan(t, planInput)

	// when version 2 commits between the initial read and migration to version 3
	interleaved := &migrateAfterSubscriptionRead{
		Service:  deps.SubscriptionService,
		workflow: deps.WorkflowService,
		plan:     p2,
	}
	deps.SubscriptionService = interleaved
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))

	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	defer clock.UnFreeze()
	response, err := svc.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:            before.Subscription.NamespacedID,
		TargetVersion: lo.ToPtr(p3.ToCreateSubscriptionPlanInput().Plan.Version),
	})
	require.NoError(t, err)

	// then current describes version 2, which the workflow actually amended
	require.Equal(t, p2.ToCreateSubscriptionPlanInput().Plan, response.Current.PlanRef)
	require.Equal(t, interleaved.migrated.Subscription, response.Current)
	require.Equal(t, p3.ToCreateSubscriptionPlanInput().Plan, response.Next.Subscription.PlanRef)
	require.Equal(t, response.Current.ID, response.Next.Subscription.ID)
}
