package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddAddon(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	type testCaseDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	runWithDeps := func(fn func(t *testing.T, deps testCaseDeps)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, testCaseDeps{deps: deps})
		}
	}

	t.Run("Should error on invalid input", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(),
				subscriptiontestutils.ExampleAddonRateCard4.Clone(),
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 0,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		expectedErr := addonInp.Validate()
		require.NotNil(t, expectedErr)

		_, _, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)

		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, expectedErr.Error())
	}))

	t.Run("Should error with not implemented if the subscription already has addons", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		_ = deps.deps.FeatureConnector.CreateExampleFeatures(t)

		// Let's create a plan
		p, err := deps.deps.PlanService.CreatePlan(context.Background(), subscriptiontestutils.GetExamplePlanInput(t))
		require.Nil(t, err)
		require.NotNil(t, p)

		// Let's create two addons that are compatible
		addonInp := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(),
			subscriptiontestutils.ExampleAddonRateCard4.Clone(),
		)

		add1 := deps.deps.AddonService.CreateTestAddon(t, addonInp)

		addonInp.Key = "some-new-key"

		add2 := deps.deps.AddonService.CreateTestAddon(t, addonInp)

		// Let's link both addons to the plan
		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add1.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add2.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's publish the plan

		p, err = deps.deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
			NamespacedID: p.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
				EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
			},
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's create a subscription from the plan

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, &plansubscription.Plan{
			Plan: p.AsProductCatalogPlan(),
			Ref:  &p.NamespacedID,
		}, now)

		// Let's add the first addon
		_ = subscriptiontestutils.CreateAddonForSubscription(t, &deps.deps, subView.Subscription.NamespacedID, add1.NamespacedID, models.CadencedModel{
			ActiveFrom: now,
			ActiveTo:   nil,
		})

		// Now let's try to add the second addon and see it fail

		_, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add2.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})

		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericNotImplementedError{}))
		require.True(t, models.IsGenericNotImplementedError(err))
	}))

	t.Run("Should sync subscription with new addons contents", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		// Let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		ogView := subView
		require.NotNil(t, ogView)

		spec := subView.AsSpec()

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		// Let's figure out what the expected spec should be
		{
			diff, err := addondiff.GetDiffableFromAddon(subView, subAdd)
			require.NoError(t, err)

			require.NoError(t, spec.Apply(diff.GetApplies(), subscription.ApplyContext{
				CurrentTime: now,
			}))
		}

		newSpec := subView.AsSpec()

		// Due to not knowing the FeatureIDs before the subscription is updated, we cannot use subscriptiontestutils.SpecsEqual properly
		// We'll strip all FeatureIDs from the comparison, which OPENS UP silent errors but this is the best we can do for now
		stripFeatureIDs(&spec)
		stripFeatureIDs(&newSpec)

		subscriptiontestutils.SpecsEqual(t, newSpec, spec)
	}))

	t.Run("Should return conflict error if subscription already has that addon purchased", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, _, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		_, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericConflictError{}))
		require.True(t, models.IsGenericConflictError(err))
	}))
}

// Instead of stripping them, we could also populate them with the correct values
func stripFeatureIDs(spec *subscription.SubscriptionSpec) {
	for _, phase := range spec.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				_ = item.RateCard.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					m.FeatureID = nil
					return m, nil
				})
			}
		}
	}
}
