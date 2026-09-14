package subscription_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestMigrationNextCycleAcrossPhaseBoundary(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given matching timelines with the next billing cycle in the second phase
	fixture := newMigrationFixtureWithPlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, migrationPlanAddingItemToBothPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when migration is scheduled for that boundary
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		Timing: &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)},
	})

	// then only the future phase gains the item and existing billing periods survive
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixture.assertSubscriptionPreserved(after, target)
	assertScheduledMigrationItems(t, fixture, after, at)
}

func TestMigrationCustomTimeInLaterPhase(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given matching timelines and a target that adds an item
	fixture := newMigrationFixtureWithPlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, migrationPlanAddingItemToBothPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when migration is scheduled beyond the next cycle, inside the second phase
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		Timing: &subscription.Timing{Custom: &at},
	})

	// then the plan reference advances now, but the new item starts at the requested time
	fixture.assertSubscriptionPreserved(after, target)
	assertScheduledMigrationItems(t, fixture, after, at)
}

func TestMigrationRejectsMisalignedCustomTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a monthly subscription and a later plan version
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when the requested date is not a billing boundary
	at := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	_, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:     fixture.before.Subscription.NamespacedID,
		Timing: &subscription.Timing{Custom: &at},
	})

	// then rejection leaves the subscription unchanged
	require.ErrorContains(t, err, "must align")
	assertMigrationViewUnchanged(t, fixture)
}

func TestMigrationRejectsPastCustomTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version and a billing boundary that has already passed
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when migration requests that past boundary
	_, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:     fixture.before.Subscription.NamespacedID,
		Timing: &subscription.Timing{Custom: &start},
	})

	// then rejection leaves the subscription unchanged
	require.ErrorContains(t, err, "in the past")
	assertMigrationViewUnchanged(t, fixture)
}

func migrationPlanWithTwoPhases(t *testing.T) plan.CreatePlanInput {
	t.Helper()

	return subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(lo.ToPtr(datetime.MustParseDuration(t, "P1M")), subscriptiontestutils.ExampleRateCard1.Clone()).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard1.Clone()).Build()
}

func migrationPlanAddingItemToBothPhases(t *testing.T) plan.CreatePlanInput {
	t.Helper()

	input := migrationPlanWithTwoPhases(t)
	for idx := range input.Phases {
		input.Phases[idx].RateCards = append(input.Phases[idx].RateCards, subscriptiontestutils.ExampleRateCard3ForAddons.Clone())
	}

	return input
}

func assertScheduledMigrationItems(t *testing.T, fixture migrationFixture, after subscription.SubscriptionView, at time.Time) {
	t.Helper()

	key := subscriptiontestutils.ExampleFeatureKey2
	require.NotContains(t, after.Phases[0].ItemsByKey, key)
	added := after.Phases[1].ItemsByKey[key]
	require.Len(t, added, 1)
	require.Equal(t, at, added[0].SubscriptionItem.ActiveFrom)
	require.NotNil(t, added[0].Entitlement)
	require.Equal(t, &at, added[0].Entitlement.Entitlement.ActiveFrom)

	for idx, phase := range fixture.before.Phases {
		require.Equal(t, phase.SubscriptionPhase, after.Phases[idx].SubscriptionPhase)
		beforeItems := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		afterItems := after.Phases[idx].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.Len(t, afterItems, len(beforeItems))
		for itemIdx, item := range beforeItems {
			require.Equal(t, item.SubscriptionItem, afterItems[itemIdx].SubscriptionItem)
			require.Equal(t, item.Spec, afterItems[itemIdx].Spec)
			require.Equal(t, item.Entitlement.Cadence, afterItems[itemIdx].Entitlement.Cadence)
		}
	}

	// Billing includes the new phase once its service period has begun.
	billingAt := at.Add(time.Second)
	beforeBilling := fixture.billingState(fixture.before, billingAt)
	afterBilling := fixture.billingState(after, billingAt)
	require.NotEmpty(t, beforeBilling.Items)

	for _, old := range beforeBilling.Items {
		actual, ok := lo.Find(afterBilling.Items, func(item targetstate.StateItem) bool {
			return item.UniqueID == old.UniqueID
		})
		require.True(t, ok, "missing unchanged billable %s", old.UniqueID)
		require.Equal(t, old.ServicePeriod, actual.ServicePeriod)
		require.Equal(t, old.FullServicePeriod, actual.FullServicePeriod)
		require.Equal(t, old.BillingPeriod, actual.BillingPeriod)
		require.Equal(t, old.GetInvoiceAt(), actual.GetInvoiceAt())
	}

	addedBilling := lo.Filter(afterBilling.Items, func(item targetstate.StateItem, _ int) bool {
		return item.Spec.ItemKey == key
	})
	require.NotEmpty(t, addedBilling)
	for _, item := range addedBilling {
		require.False(t, item.ServicePeriod.From.Before(at))
	}
}

func assertMigrationViewUnchanged(t *testing.T, fixture migrationFixture) {
	t.Helper()

	after, err := fixture.deps.SubscriptionService.GetView(t.Context(), fixture.before.Subscription.NamespacedID)
	require.NoError(t, err)
	require.Equal(t, fixture.before.Subscription, after.Subscription)
	require.Equal(t, fixture.before.Spec, after.Spec)
}

func TestMigrationScheduledPriceChangeInLaterPhase(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a target that reprices the second phase
	fixture := newMigrationFixtureWithPlan(t, migrationPlanWithTwoPhases(t))
	next := migrationPlanWithTwoPhases(t)
	require.NoError(t, next.Phases[1].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(50)})
		return meta, nil
	}))

	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, next)
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when the price change is scheduled inside that later phase
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		Timing: &subscription.Timing{Custom: &at},
	})

	// then phase one survives and phase two keeps its old price until the requested boundary
	fixture.assertSubscriptionPreserved(after, target)
	require.Equal(t, fixture.before.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].SubscriptionItem, after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].SubscriptionItem)
	old := fixture.before.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].SubscriptionItem
	items := after.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
	require.Len(t, items, 2)
	require.Equal(t, old.ActiveFrom, items[0].SubscriptionItem.ActiveFrom)
	require.Equal(t, &at, items[0].SubscriptionItem.ActiveTo)
	require.Equal(t, at, items[1].SubscriptionItem.ActiveFrom)

	beforePrice, err := items[0].SubscriptionItem.RateCard.AsMeta().Price.AsUnit()
	require.NoError(t, err)
	require.Equal(t, 100.0, beforePrice.Amount.InexactFloat64())
	afterPrice, err := items[1].SubscriptionItem.RateCard.AsMeta().Price.AsUnit()
	require.NoError(t, err)
	require.Equal(t, 50.0, afterPrice.Amount.InexactFloat64())

	// The affected future item can be rematerialized; its earlier billing periods must survive.
	beforeBilling := fixture.billingState(fixture.before, at.Add(time.Second))
	afterBilling := fixture.billingState(after, at.Add(time.Second))
	earlier := lo.Filter(beforeBilling.Items, func(item targetstate.StateItem, _ int) bool {
		return !item.ServicePeriod.To.After(at)
	})
	require.NotEmpty(t, earlier)

	for _, expected := range earlier {
		actual, ok := lo.Find(afterBilling.Items, func(item targetstate.StateItem) bool {
			return item.UniqueID == expected.UniqueID
		})
		require.True(t, ok, "missing earlier billable %s", expected.UniqueID)
		require.Equal(t, expected.ServicePeriod, actual.ServicePeriod)
		require.Equal(t, expected.FullServicePeriod, actual.FullServicePeriod)
		require.Equal(t, expected.BillingPeriod, actual.BillingPeriod)
		require.Equal(t, expected.GetInvoiceAt(), actual.GetInvoiceAt())
	}
}
