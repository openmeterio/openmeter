package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestUpdateMaterializesLegacySubscriptionItemCurrency(t *testing.T) {
	// given:
	// - a subscription whose priced item represents a pre-backfill row with no stored currency
	// when:
	// - the subscription goes through the normal update path
	// then:
	// - the recreated item stores the subscription's existing currency without changing subscription semantics
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	deps, legacyView, legacyItemID := createSubscriptionWithLegacyPricedItemCurrency(t, now)

	_, err := deps.SubscriptionService.Update(t.Context(), legacyView.Subscription.NamespacedID, legacyView.AsSpec())
	require.NoError(t, err)

	requireMaterializedSubscriptionItemCurrency(t, deps, legacyView.Subscription.NamespacedID, legacyItemID)
}

func TestCancelMaterializesLegacySubscriptionItemCurrency(t *testing.T) {
	// given:
	// - a subscription whose priced item represents a pre-backfill row with no stored currency
	// when:
	// - cancellation reconciles the subscription directly through sync
	// then:
	// - the recreated item stores the subscription's existing currency
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	deps, legacyView, legacyItemID := createSubscriptionWithLegacyPricedItemCurrency(t, now)
	nextBillingCycle := subscription.TimingNextBillingCycle

	_, err := deps.SubscriptionService.Cancel(t.Context(), legacyView.Subscription.NamespacedID, subscription.Timing{
		Enum: &nextBillingCycle,
	})
	require.NoError(t, err)

	requireMaterializedSubscriptionItemCurrency(t, deps, legacyView.Subscription.NamespacedID, legacyItemID)
}

func createSubscriptionWithLegacyPricedItemCurrency(t *testing.T, now time.Time) (subscriptiontestutils.SubscriptionDependencies, subscription.SubscriptionView, string) {
	t.Helper()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)
	t.Cleanup(func() { dbDeps.Cleanup(t) })

	deps := subscriptiontestutils.NewService(t, dbDeps)
	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).
		Build()
	planInput.Key = "legacy-subscription-item-currency"
	plan := deps.PlanHelper.CreatePlan(t, planInput)
	view := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, now)

	items := view.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()]
	require.Len(t, items, 1)
	storedCurrency := items[0].SubscriptionItem.RateCard.AsMeta().Currency
	require.NotNil(t, storedCurrency)
	require.Equal(t, currencyx.Code("USD"), storedCurrency.GetCode())

	legacyItemID := items[0].SubscriptionItem.ID
	_, err := dbDeps.DBClient.SubscriptionItem.
		UpdateOneID(legacyItemID).
		ClearFiatCurrencyCode().
		Save(t.Context())
	require.NoError(t, err)

	legacyView, err := deps.SubscriptionService.GetView(t.Context(), view.Subscription.NamespacedID)
	require.NoError(t, err)
	legacyItems := legacyView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()]
	require.Len(t, legacyItems, 1)
	require.Nil(t, legacyItems[0].SubscriptionItem.RateCard.AsMeta().Currency)

	return deps, legacyView, legacyItemID
}

func requireMaterializedSubscriptionItemCurrency(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies, subscriptionID models.NamespacedID, legacyItemID string) {
	t.Helper()

	updatedView, err := deps.SubscriptionService.GetView(t.Context(), subscriptionID)
	require.NoError(t, err)
	updatedItems := updatedView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()]
	require.Len(t, updatedItems, 1)
	require.NotEqual(t, legacyItemID, updatedItems[0].SubscriptionItem.ID)

	updatedCurrency := updatedItems[0].SubscriptionItem.RateCard.AsMeta().Currency
	require.NotNil(t, updatedCurrency)
	require.Equal(t, currencyx.Code("USD"), updatedCurrency.GetCode())
}
