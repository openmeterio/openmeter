package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
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
		_, subView := subscriptiontestutils.CreateSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t), now)

		add := deps.deps.AddonService.CreateTestAddon(t, subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
			subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
		))

		addonInp := subscriptionaddon.CreateSubscriptionAddonInput{
			AddonID:        add.ID,
			SubscriptionID: subView.Subscription.NamespacedID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   0,
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
		_, subView := subscriptiontestutils.CreateSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t), now)

		addInp := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(),
			subscriptiontestutils.ExampleAddonRateCard4.Clone(),
		)
		_, _ = subscriptiontestutils.CreateAddonForSub(t, &deps.deps, subView.Subscription.NamespacedID, addInp)

		addInp.Key = "some-new-key"

		// We need a new addon to avoid conflicts
		add := deps.deps.AddonService.CreateTestAddon(t, addInp)

		_, _, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionaddon.CreateSubscriptionAddonInput{
			AddonID:        add.ID,
			SubscriptionID: subView.Subscription.NamespacedID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   1,
			},
		})

		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericNotImplementedError{}))
		require.True(t, models.IsGenericNotImplementedError(err))
	}))

	t.Run("Should sync subscription with new addons contents", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		_, subView := subscriptiontestutils.CreateSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t), now)

		ogView := subView
		require.NotNil(t, ogView)

		spec := subView.AsSpec()

		add := deps.deps.AddonService.CreateTestAddon(t, subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
			subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
		))

		addonInp := subscriptionaddon.CreateSubscriptionAddonInput{
			AddonID:        add.ID,
			SubscriptionID: subView.Subscription.NamespacedID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   1,
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
		_, subView := subscriptiontestutils.CreateSubFromPlan(t, &deps.deps, subscriptiontestutils.GetExamplePlanInput(t), now)

		add := deps.deps.AddonService.CreateTestAddon(t, subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
			subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
		))

		addonInp := subscriptionaddon.CreateSubscriptionAddonInput{
			AddonID:        add.ID,
			SubscriptionID: subView.Subscription.NamespacedID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   1,
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
