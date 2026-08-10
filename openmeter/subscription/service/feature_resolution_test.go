package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestCreateResolvesFeatureOnlyItems(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	t.Run("resolves and pins a feature supplied by key", func(t *testing.T) {
		// given: a structurally valid feature-only subscription item with no pinned feature ID
		clock.FreezeTime(now)
		defer clock.UnFreeze()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		features := deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
		customer := deps.CustomerAdapter.CreateExampleCustomer(t)

		spec, err := subscriptiontestutils.BuildTestSubscriptionSpec(t).
			AddPhase(nil, newFeatureOnlyRateCard(t, subscriptiontestutils.ExampleFeatureKey, nil)).
			Build()
		require.NoError(t, err)
		spec.CustomerId = customer.ID
		spec.Plan = nil

		// when: the subscription is created through the core service
		sub, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
		require.NoError(t, err)

		// then: the feature-only item is materialized without creating an entitlement
		items, err := deps.ItemRepo.GetForSubscriptionAt(t.Context(), subscription.GetForSubscriptionAtInput{
			Namespace:      sub.Namespace,
			SubscriptionID: sub.ID,
			At:             now,
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Nil(t, items[0].EntitlementID)
		require.Equal(t, subscriptiontestutils.ExampleFeatureKey, lo.FromPtr(items[0].RateCard.GetFeatureKey()))

		view, err := deps.SubscriptionService.GetView(t.Context(), sub.NamespacedID)
		require.NoError(t, err)
		itemView := view.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0]
		require.NotNil(t, itemView.Feature)
		require.Equal(t, features[0].ID, itemView.Feature.ID)
	})

	t.Run("rejects an unknown feature key", func(t *testing.T) {
		// given: a feature-only subscription item referencing a missing feature
		clock.FreezeTime(now)
		defer clock.UnFreeze()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		customer := deps.CustomerAdapter.CreateExampleCustomer(t)

		const missingFeatureKey = "missing-feature"
		spec, err := subscriptiontestutils.BuildTestSubscriptionSpec(t).
			AddPhase(nil, newFeatureOnlyRateCard(t, missingFeatureKey, nil)).
			Build()
		require.NoError(t, err)
		spec.CustomerId = customer.ID
		spec.Plan = nil

		// when: subscription creation validates its complete target spec
		_, err = deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)

		// then: the write is rejected before materialization
		require.ErrorIs(t, err, productcatalog.ErrRateCardFeatureNotFound)
	})

	t.Run("rejects mismatched feature identifiers", func(t *testing.T) {
		// given: a feature-only item whose key and ID identify different features
		clock.FreezeTime(now)
		defer clock.UnFreeze()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		features := deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
		customer := deps.CustomerAdapter.CreateExampleCustomer(t)

		spec, err := subscriptiontestutils.BuildTestSubscriptionSpec(t).
			AddPhase(nil, newFeatureOnlyRateCard(t, subscriptiontestutils.ExampleFeatureKey, &features[1].ID)).
			Build()
		require.NoError(t, err)
		spec.CustomerId = customer.ID
		spec.Plan = nil

		// when: subscription creation resolves both references
		_, err = deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)

		// then: the conflicting identity is rejected
		require.ErrorIs(t, err, productcatalog.ErrRateCardFeatureMismatch)
	})
}

func TestEditRunningResolvesFeatureOnlyItems(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	immediate := subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}

	t.Run("resolves and pins an added feature-only item", func(t *testing.T) {
		// given: a running subscription and an existing feature referenced only by key
		clock.FreezeTime(now)
		defer clock.UnFreeze()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		features := deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
		customer := deps.CustomerAdapter.CreateExampleCustomer(t)

		spec, err := subscriptiontestutils.BuildTestSubscriptionSpec(t).
			AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).
			Build()
		require.NoError(t, err)
		spec.CustomerId = customer.ID
		spec.Plan = nil

		sub, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
		require.NoError(t, err)

		// when: a running edit adds a feature-only item without a pinned feature ID
		_, err = deps.WorkflowService.EditRunning(t.Context(), sub.NamespacedID, []subscription.Patch{
			patch.PatchAddItem{
				PhaseKey: "test_phase_1",
				ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: "test_phase_1",
							ItemKey:  subscriptiontestutils.ExampleFeatureKey,
							RateCard: newFeatureOnlyRateCard(t, subscriptiontestutils.ExampleFeatureKey, nil),
						},
					},
				},
			},
		}, immediate)
		require.NoError(t, err)

		// then: the new feature-only item is materialized without an entitlement
		items, err := deps.ItemRepo.GetForSubscriptionAt(t.Context(), subscription.GetForSubscriptionAtInput{
			Namespace:      sub.Namespace,
			SubscriptionID: sub.ID,
			At:             now,
		})
		require.NoError(t, err)
		item, found := lo.Find(items, func(item subscription.SubscriptionItem) bool {
			return item.Key == subscriptiontestutils.ExampleFeatureKey
		})
		require.True(t, found)
		require.Nil(t, item.EntitlementID)

		view, err := deps.SubscriptionService.GetView(t.Context(), sub.NamespacedID)
		require.NoError(t, err)
		itemView := view.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0]
		require.NotNil(t, itemView.Feature)
		require.Equal(t, features[0].ID, itemView.Feature.ID)
	})

	t.Run("rejects an added item with an unknown feature key", func(t *testing.T) {
		// given: a running subscription and a feature-only item referencing a missing feature
		clock.FreezeTime(now)
		defer clock.UnFreeze()

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		customer := deps.CustomerAdapter.CreateExampleCustomer(t)

		spec, err := subscriptiontestutils.BuildTestSubscriptionSpec(t).
			AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).
			Build()
		require.NoError(t, err)
		spec.CustomerId = customer.ID
		spec.Plan = nil

		sub, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
		require.NoError(t, err)

		const missingFeatureKey = "missing-feature"

		// when: the running edit adds the invalid item
		_, err = deps.WorkflowService.EditRunning(t.Context(), sub.NamespacedID, []subscription.Patch{
			patch.PatchAddItem{
				PhaseKey: "test_phase_1",
				ItemKey:  missingFeatureKey,
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: "test_phase_1",
							ItemKey:  missingFeatureKey,
							RateCard: newFeatureOnlyRateCard(t, missingFeatureKey, nil),
						},
					},
				},
			},
		}, immediate)

		// then: the edit fails and the persisted subscription remains unchanged
		require.ErrorIs(t, err, productcatalog.ErrRateCardFeatureNotFound)

		items, err := deps.ItemRepo.GetForSubscriptionAt(t.Context(), subscription.GetForSubscriptionAtInput{
			Namespace:      sub.Namespace,
			SubscriptionID: sub.ID,
			At:             now,
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subscriptiontestutils.ExampleRateCard2.Key(), items[0].Key)
	})
}

func newFeatureOnlyRateCard(t *testing.T, featureKey string, featureID *string) productcatalog.RateCard {
	t.Helper()

	rateCard := subscriptiontestutils.ExampleRateCard5ForAddons.Clone()
	err := rateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = featureKey
		meta.FeatureKey = lo.ToPtr(featureKey)
		meta.FeatureID = featureID
		return meta, nil
	})
	require.NoError(t, err)

	return rateCard
}
