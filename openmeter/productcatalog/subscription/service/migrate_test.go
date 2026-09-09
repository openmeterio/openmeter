package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
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

func TestMigratePreservesUnaffectedItems(t *testing.T) {
	for _, scenario := range []string{"add item", "change price", "remove item", "metadata only", "next cycle"} {
		t.Run(scenario, func(t *testing.T) {
			// given an active subscription with two independent rate cards
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			ctx := t.Context()
			db := subscriptiontestutils.SetupDBDeps(t)
			defer db.Cleanup(t)
			deps := subscriptiontestutils.NewService(t, db)
			deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
				subscriptiontestutils.ExampleRateCard1.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build()
			p1 := deps.PlanHelper.CreatePlan(t, input)
			before, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				Namespace: cust.Namespace, CustomerID: cust.ID,
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
				},
			}, p1)
			require.NoError(t, err)
			at := start.Add(10 * 24 * time.Hour)
			clock.FreezeTime(at.Add(-time.Second))
			nextInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
				subscriptiontestutils.ExampleRateCard1.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build()
			switch scenario {
			case "add item", "next cycle":
				nextInput.Phases[0].RateCards = append(nextInput.Phases[0].RateCards, subscriptiontestutils.ExampleRateCard3ForAddons.Clone())
			case "change price":
				require.NoError(t, nextInput.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(50)})
					return meta, nil
				}))
			case "remove item":
				nextInput.Phases[0].RateCards = nextInput.Phases[0].RateCards[1:]
			case "metadata only":
				nextInput.Name = "new catalog name"
			}
			p2 := deps.PlanHelper.CreatePlan(t, nextInput)
			clock.FreezeTime(at)
			svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
			request := plansubscription.MigrateSubscriptionRequest{ID: before.Subscription.NamespacedID}
			if scenario == "next cycle" {
				request.Timing = &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}
				at = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			}
			// when migrating the offering in the middle of a billing period
			response, err := svc.Migrate(ctx, request)
			require.NoError(t, err)
			after := response.Next
			// then the subscription and unaffected item/entitlement identities survive
			require.Equal(t, before.Subscription.ID, after.Subscription.ID)
			require.Equal(t, before.Subscription.ActiveFrom, after.Subscription.ActiveFrom)
			require.Nil(t, after.Subscription.ActiveTo)
			require.Equal(t, before.Subscription.BillingAnchor, after.Subscription.BillingAnchor)
			require.Equal(t, p2.ToCreateSubscriptionPlanInput().Plan, after.Subscription.PlanRef)
			require.Equal(t, before.Phases[0].SubscriptionPhase.ID, after.Phases[0].SubscriptionPhase.ID)
			require.Equal(t, before.Phases[0].ItemsByKey["rate-card-2"], after.Phases[0].ItemsByKey["rate-card-2"])
			key := subscriptiontestutils.ExampleFeatureKey
			switch scenario {
			case "change price":
				items := after.Phases[0].ItemsByKey[key]
				require.Len(t, items, 2)
				require.Equal(t, at, *items[0].SubscriptionItem.ActiveTo)
				require.Equal(t, at, items[1].SubscriptionItem.ActiveFrom)
				require.Equal(t, start, items[0].SubscriptionItem.ActiveFrom)
			case "remove item":
				items := after.Phases[0].ItemsByKey[key]
				require.Len(t, items, 1)
				require.Equal(t, at, *items[0].SubscriptionItem.ActiveTo)
			default:
				require.Len(t, after.Phases[0].ItemsByKey[key], 1)
				require.Equal(t, before.Phases[0].ItemsByKey[key][0].SubscriptionItem.ID, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.ID)
				require.Equal(t, before.Phases[0].ItemsByKey[key][0].SubscriptionItem.EntitlementID, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.EntitlementID)
				require.Equal(t, before.Phases[0].ItemsByKey[key][0].SubscriptionItem.CadencedModel, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.CadencedModel)
			}
			if scenario == "add item" || scenario == "next cycle" {
				added := after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2]
				require.Len(t, added, 1)
				require.Equal(t, at, added[0].SubscriptionItem.ActiveFrom)
			}
			require.NoError(t, after.Validate(true))
			// An editor holding the previous plan's spec cannot undo a migration.
			_, err = deps.SubscriptionService.Update(ctx, before.Subscription.NamespacedID, before.Spec)
			require.Error(t, err)

			builder := targetstate.NewBuilder(testutils.NewLogger(t), noop.NewTracerProvider().Tracer("test"))
			beforeBilling, err := builder.Build(ctx, targetstate.BuildInput{Persisted: persistedstate.State{ByUniqueID: map[string]persistedstate.Item{}, Invoices: persistedstate.Invoices{}}, AsOf: at, SubscriptionView: &before})
			require.NoError(t, err)
			afterBilling, err := builder.Build(ctx, targetstate.BuildInput{Persisted: persistedstate.State{ByUniqueID: map[string]persistedstate.Item{}, Invoices: persistedstate.Invoices{}}, AsOf: at, SubscriptionView: &after})
			require.NoError(t, err)
			// Unaffected billables keep their reconciliation identity, invoice time,
			// and complete service/billing periods across migration.
			for _, old := range beforeBilling.Items {
				if old.Spec.ItemKey == key && (scenario == "change price" || scenario == "remove item") {
					continue
				}
				found, ok := lo.Find(afterBilling.Items, func(item targetstate.StateItem) bool { return item.UniqueID == old.UniqueID })
				require.True(t, ok, "missing billable %s", old.UniqueID)
				require.Equal(t, old.ServicePeriod, found.ServicePeriod)
				require.Equal(t, old.FullServicePeriod, found.FullServicePeriod)
				require.Equal(t, old.BillingPeriod, found.BillingPeriod)
				require.Equal(t, old.GetInvoiceAt(), found.GetInvoiceAt())
			}
		})
	}
}
