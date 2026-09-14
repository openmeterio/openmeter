package subscription_test

import (
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
)

func TestMigrationAdditionPreservesExistingItems(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with an additional item
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()
	nextPlan.Phases[0].RateCards = append(nextPlan.Phases[0].RateCards, subscriptiontestutils.ExampleRateCard3ForAddons.Clone())

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when the subscription is migrated mid-cycle
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{})

	// then the new item starts at migration and both existing items keep their billing periods
	at := start.Add(10 * 24 * time.Hour)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)
	fixture.assertItemAdded(after, subscriptiontestutils.ExampleFeatureKey2, at)
	fixture.assertItemUnchanged(after, subscriptiontestutils.ExampleFeatureKey, at)

	fixture.assertStalePlanEditRejected()
}

func TestMigrationPriceChangeSplitsOnlyChangedItem(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with a changed price
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()
	require.NoError(t, nextPlan.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(50)})
		return meta, nil
	}))

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when the subscription is migrated mid-cycle
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{})

	// then only the repriced item is split at migration
	at := start.Add(10 * 24 * time.Hour)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)

	items := after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
	require.Len(t, items, 2)
	require.Equal(t, start, items[0].SubscriptionItem.ActiveFrom)
	require.Equal(t, at, *items[0].SubscriptionItem.ActiveTo)
	require.Equal(t, at, items[1].SubscriptionItem.ActiveFrom)

	originalPrice, err := items[0].SubscriptionItem.RateCard.AsMeta().Price.AsUnit()
	require.NoError(t, err)
	require.Equal(t, 100.0, originalPrice.Amount.InexactFloat64())

	replacementPrice, err := items[1].SubscriptionItem.RateCard.AsMeta().Price.AsUnit()
	require.NoError(t, err)
	require.Equal(t, 50.0, replacementPrice.Amount.InexactFloat64())

	fixture.assertStalePlanEditRejected()
}

func TestMigrationRemovalEndsOnlyRemovedItem(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version without the first item
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()
	nextPlan.Phases[0].RateCards = nextPlan.Phases[0].RateCards[1:]

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when the subscription is migrated mid-cycle
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{})

	// then only the removed item ends at migration
	at := start.Add(10 * 24 * time.Hour)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)

	items := after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
	require.Len(t, items, 1)
	require.Equal(t, at, *items[0].SubscriptionItem.ActiveTo)

	fixture.assertStalePlanEditRejected()
}

func TestMigrationMetadataChangePreservesItems(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with only a name change
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()
	nextPlan.Name = "new catalog name"

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when the subscription is migrated mid-cycle
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{})

	// then the plan reference advances without changing either item
	at := start.Add(10 * 24 * time.Hour)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)
	fixture.assertItemUnchanged(after, subscriptiontestutils.ExampleFeatureKey, at)

	fixture.assertStalePlanEditRejected()
}

func TestMigrationNextCycleAdditionPreservesExistingItems(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with an additional item
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()
	nextPlan.Phases[0].RateCards = append(nextPlan.Phases[0].RateCards, subscriptiontestutils.ExampleRateCard3ForAddons.Clone())

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when migration is requested for the next billing cycle
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		Timing: &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)},
	})

	// then the added item starts next cycle and existing billing periods stay intact
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)
	fixture.assertItemAdded(after, subscriptiontestutils.ExampleFeatureKey2, at)
	fixture.assertItemUnchanged(after, subscriptiontestutils.ExampleFeatureKey, at)

	fixture.assertStalePlanEditRejected()
}

func TestMigrationSameAnchorPreservesItems(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with unchanged items
	fixture := newMigrationFixture(t)

	nextPlan := fixture.newPlanVersion()

	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, nextPlan)

	// when migration supplies the existing anchor in a different time zone
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		BillingAnchor: lo.ToPtr(start.In(time.FixedZone("UTC+2", 2*60*60))),
	})

	// then the existing anchor and both items are preserved
	at := start.Add(10 * 24 * time.Hour)
	fixture.assertSubscriptionPreserved(after, target)
	fixture.assertItemUnchanged(after, "rate-card-2", at)
	fixture.assertItemUnchanged(after, subscriptiontestutils.ExampleFeatureKey, at)

	fixture.assertStalePlanEditRejected()
}

func TestMigrationChangedAnchorCreatesReplacement(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version and a running subscription
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())

	// when migration moves billing to the fourth day of the month
	at := start.Add(10 * 24 * time.Hour)
	anchor := start.Add(3 * 24 * time.Hour)
	clock.FreezeTime(at)
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		BillingAnchor: &anchor,
	})

	// then the original subscription ends and the replacement uses the new anchor
	fixture.assertSubscriptionReplaced(after, target, at)
	require.Equal(t, anchor, after.Subscription.BillingAnchor)
	fixture.assertBillingPeriodEndsAt(after, at, time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC))
}

func TestMigrationFutureAnchorIsPreserved(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version and a running subscription
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())

	// when the supplied anchor is later than the migration time
	at := start.Add(10 * 24 * time.Hour)
	anchor := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(at)
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		BillingAnchor: &anchor,
	})

	// then billing uses the anchor's recurrence without moving the stored anchor
	fixture.assertSubscriptionReplaced(after, target, at)
	require.Equal(t, anchor, after.Subscription.BillingAnchor)
	fixture.assertBillingPeriodEndsAt(after, at, time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC))
}

func TestMigrationAnchorChangeAtNextCycle(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a subscription billed on the first day of each month
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())

	// when an anchor change is requested for the next billing cycle
	anchor := start.Add(3 * 24 * time.Hour)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		BillingAnchor: &anchor,
		Timing:        &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)},
	})

	// then the old anchor determines the change time and the new one determines billing
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixture.assertSubscriptionReplaced(after, target, at)
	require.Equal(t, anchor, after.Subscription.BillingAnchor)
	fixture.assertBillingPeriodEndsAt(after, at, time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC))
}

func TestMigrationInvalidAnchorLeavesSubscriptionIntact(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version and a running subscription
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())

	// when migration supplies an invalid anchor
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	_, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:            fixture.before.Subscription.NamespacedID,
		BillingAnchor: lo.ToPtr(time.Time{}),
	})
	require.Error(t, err)

	// then cancellation is rolled back along with the failed replacement
	current, err := fixture.deps.SubscriptionService.Get(t.Context(), fixture.before.Subscription.NamespacedID)
	require.NoError(t, err)
	require.Nil(t, current.ActiveTo)
	require.Equal(t, fixture.before.Subscription.PlanRef, current.PlanRef)
	require.Equal(t, fixture.before.Subscription.BillingAnchor, current.BillingAnchor)
}

type migrationFixture struct {
	t      *testing.T
	deps   testDeps
	before subscription.SubscriptionView
}

func newMigrationFixture(t *testing.T) migrationFixture {
	t.Helper()

	input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
		subscriptiontestutils.ExampleRateCard1.Clone(),
		subscriptiontestutils.ExampleRateCard2.Clone(),
	).Build()

	return newMigrationFixtureWithPlan(t, input)
}

func newMigrationFixtureWithPlan(t *testing.T, input plan.CreatePlanInput) migrationFixture {
	t.Helper()

	deps := setup(t, setupConfig{})
	t.Cleanup(func() { deps.cleanup(t) })

	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)

	original := deps.PlanHelper.CreatePlan(t, input)

	before, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  customer.Namespace,
		CustomerID: customer.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
		},
	}, original)
	require.NoError(t, err)

	return migrationFixture{t: t, deps: deps, before: before}
}

func (f migrationFixture) newPlanVersion() plan.CreatePlanInput {
	f.t.Helper()

	return subscriptiontestutils.BuildTestPlanInput(f.t).AddPhase(nil,
		subscriptiontestutils.ExampleRateCard1.Clone(),
		subscriptiontestutils.ExampleRateCard2.Clone(),
	).Build()
}

func (f migrationFixture) migrate(request plansubscription.MigrateSubscriptionRequest) subscription.SubscriptionView {
	f.t.Helper()

	request.ID = f.before.Subscription.NamespacedID
	response, err := f.deps.pcSubscriptionService.Migrate(f.t.Context(), request)
	require.NoError(f.t, err)
	require.NoError(f.t, response.Next.Validate(true))

	return response.Next
}

func (f migrationFixture) assertSubscriptionPreserved(after subscription.SubscriptionView, target subscription.Plan) {
	f.t.Helper()

	require.Equal(f.t, f.before.Subscription.ID, after.Subscription.ID)
	require.Equal(f.t, f.before.Subscription.ActiveFrom, after.Subscription.ActiveFrom)
	require.Nil(f.t, after.Subscription.ActiveTo)
	require.Equal(f.t, f.before.Subscription.BillingAnchor, after.Subscription.BillingAnchor)
	require.Equal(f.t, target.ToCreateSubscriptionPlanInput().Plan, after.Subscription.PlanRef)
	require.Equal(f.t, f.before.Phases[0].SubscriptionPhase.ID, after.Phases[0].SubscriptionPhase.ID)
}

func (f migrationFixture) assertItemAdded(after subscription.SubscriptionView, key string, at time.Time) {
	f.t.Helper()

	items := after.Phases[0].ItemsByKey[key]
	require.Len(f.t, items, 1)
	require.Equal(f.t, at, items[0].SubscriptionItem.ActiveFrom)
}

// Unchanged items keep their identities, entitlements, and billing periods.
func (f migrationFixture) assertItemUnchanged(after subscription.SubscriptionView, key string, at time.Time) {
	f.t.Helper()

	beforeItems := f.before.Phases[0].ItemsByKey[key]
	afterItems := after.Phases[0].ItemsByKey[key]
	require.Len(f.t, beforeItems, 1)
	require.Len(f.t, afterItems, 1)
	require.Equal(f.t, beforeItems[0].SubscriptionItem, afterItems[0].SubscriptionItem)
	require.Equal(f.t, beforeItems[0].Spec, afterItems[0].Spec)

	beforeBilling := f.billingState(f.before, at)
	afterBilling := f.billingState(after, at)
	billables := lo.Filter(beforeBilling.Items, func(item targetstate.StateItem, _ int) bool {
		return item.Spec.ItemKey == key
	})
	require.NotEmpty(f.t, billables, "no billing periods for item %s", key)

	for _, old := range billables {
		found, ok := lo.Find(afterBilling.Items, func(item targetstate.StateItem) bool {
			return item.UniqueID == old.UniqueID
		})
		require.True(f.t, ok, "missing billable %s", old.UniqueID)

		require.Equal(f.t, old.ServicePeriod, found.ServicePeriod)
		require.Equal(f.t, old.FullServicePeriod, found.FullServicePeriod)
		require.Equal(f.t, old.BillingPeriod, found.BillingPeriod)
		require.Equal(f.t, old.GetInvoiceAt(), found.GetInvoiceAt())
	}
}

func (f migrationFixture) assertStalePlanEditRejected() {
	f.t.Helper()

	_, err := f.deps.SubscriptionService.Update(f.t.Context(), f.before.Subscription.NamespacedID, f.before.Spec)
	require.Error(f.t, err)
}

func (f migrationFixture) billingState(view subscription.SubscriptionView, at time.Time) targetstate.State {
	f.t.Helper()

	builder := targetstate.NewBuilder(testutils.NewLogger(f.t), noop.NewTracerProvider().Tracer("test"))
	state, err := builder.Build(f.t.Context(), targetstate.BuildInput{
		Persisted: persistedstate.State{
			ByUniqueID: map[string]persistedstate.Item{},
			Invoices:   persistedstate.Invoices{},
		},
		AsOf:             at,
		SubscriptionView: &view,
	})
	require.NoError(f.t, err)

	return state
}

func (f migrationFixture) assertSubscriptionReplaced(after subscription.SubscriptionView, target subscription.Plan, at time.Time) {
	f.t.Helper()

	current, err := f.deps.SubscriptionService.Get(f.t.Context(), f.before.Subscription.NamespacedID)
	require.NoError(f.t, err)
	require.Equal(f.t, &at, current.ActiveTo)
	require.Equal(f.t, f.before.Subscription.PlanRef, current.PlanRef)

	require.NotEqual(f.t, current.ID, after.Subscription.ID)
	require.Equal(f.t, at, after.Subscription.ActiveFrom)
	require.Equal(f.t, target.ToCreateSubscriptionPlanInput().Plan, after.Subscription.PlanRef)
	require.Equal(f.t, current.CustomerId, after.Subscription.CustomerId)
	require.Equal(f.t, current.MetadataModel, after.Subscription.MetadataModel)
	require.Equal(f.t, current.Name, after.Subscription.Name)
	require.Equal(f.t, current.Description, after.Subscription.Description)
	require.Equal(f.t, current.CostBasisMode, after.Subscription.CostBasisMode)

	require.Equal(f.t, &current.ID, subscription.AnnotationParser.GetPreviousSubscriptionID(after.Subscription.Annotations))
	require.Equal(f.t, &after.Subscription.ID, subscription.AnnotationParser.GetSupersedingSubscriptionID(current.Annotations))
}

func (f migrationFixture) assertBillingPeriodEndsAt(after subscription.SubscriptionView, at, end time.Time) {
	f.t.Helper()

	period, err := after.Spec.GetAlignedBillingPeriodAt(at)
	require.NoError(f.t, err)
	require.Equal(f.t, at, period.From)
	require.Equal(f.t, end, period.To)
}
