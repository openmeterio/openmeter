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
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type migrationFixture struct {
	deps      testDeps
	before    subscription.SubscriptionView
	target    subscription.Plan
	request   plansubscription.MigrateSubscriptionRequest
	scenario  string
	start, at time.Time
}

func TestMigrationPreservesUnaffectedItems(t *testing.T) {
	for _, scenario := range []string{"add item", "change price", "remove item", "metadata only", "next cycle", "deprecated anchor"} {
		t.Run(scenario, func(t *testing.T) {
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			// given a running subscription and a later plan version
			fixture := setupMigrationScenario(t, scenario, start)
			// when the subscription is migrated
			after := executeMigration(t, fixture)
			// then unchanged items keep their identity and billing periods
			assertMigrationItems(t, fixture, after)
			assertUnchangedMigrationBilling(t, fixture, after)
			assertStalePlanEditRejected(t, fixture)
		})
	}
}

func setupMigrationScenario(t *testing.T, scenario string, start time.Time) migrationFixture {
	t.Helper()
	ctx := t.Context()
	deps := setup(t, setupConfig{})
	t.Cleanup(func() { deps.cleanup(t) })
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
	target := deps.PlanHelper.CreatePlan(t, nextInput)
	clock.FreezeTime(at)

	request := plansubscription.MigrateSubscriptionRequest{ID: before.Subscription.NamespacedID}
	if scenario == "deprecated anchor" {
		request.BillingAnchor = lo.ToPtr(start.Add(3 * 24 * time.Hour))
	}
	if scenario == "next cycle" {
		request.Timing = &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}
		at = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	}

	return migrationFixture{deps: deps, before: before, target: target, request: request, scenario: scenario, start: start, at: at}
}

func executeMigration(t *testing.T, fixture migrationFixture) subscription.SubscriptionView {
	t.Helper()
	response, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), fixture.request)
	require.NoError(t, err)
	require.NoError(t, response.Next.Validate(true))
	return response.Next
}

func assertMigrationItems(t *testing.T, fixture migrationFixture, after subscription.SubscriptionView) {
	t.Helper()
	// then the subscription and unaffected item/entitlement identities survive
	require.Equal(t, fixture.before.Subscription.ID, after.Subscription.ID)
	require.Equal(t, fixture.before.Subscription.ActiveFrom, after.Subscription.ActiveFrom)
	require.Nil(t, after.Subscription.ActiveTo)
	require.Equal(t, fixture.before.Subscription.BillingAnchor, after.Subscription.BillingAnchor)
	require.Equal(t, fixture.target.ToCreateSubscriptionPlanInput().Plan, after.Subscription.PlanRef)
	require.Equal(t, fixture.before.Phases[0].SubscriptionPhase.ID, after.Phases[0].SubscriptionPhase.ID)
	require.Equal(t, fixture.before.Phases[0].ItemsByKey["rate-card-2"], after.Phases[0].ItemsByKey["rate-card-2"])
	key := subscriptiontestutils.ExampleFeatureKey
	switch fixture.scenario {
	case "change price":
		items := after.Phases[0].ItemsByKey[key]
		require.Len(t, items, 2)
		require.Equal(t, fixture.at, *items[0].SubscriptionItem.ActiveTo)
		require.Equal(t, fixture.at, items[1].SubscriptionItem.ActiveFrom)
		require.Equal(t, fixture.start, items[0].SubscriptionItem.ActiveFrom)
	case "remove item":
		items := after.Phases[0].ItemsByKey[key]
		require.Len(t, items, 1)
		require.Equal(t, fixture.at, *items[0].SubscriptionItem.ActiveTo)
	default:
		require.Len(t, after.Phases[0].ItemsByKey[key], 1)
		require.Equal(t, fixture.before.Phases[0].ItemsByKey[key][0].SubscriptionItem.ID, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.ID)
		require.Equal(t, fixture.before.Phases[0].ItemsByKey[key][0].SubscriptionItem.EntitlementID, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.EntitlementID)
		require.Equal(t, fixture.before.Phases[0].ItemsByKey[key][0].SubscriptionItem.CadencedModel, after.Phases[0].ItemsByKey[key][0].SubscriptionItem.CadencedModel)
	}
	if fixture.scenario == "add item" || fixture.scenario == "next cycle" {
		added := after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2]
		require.Len(t, added, 1)
		require.Equal(t, fixture.at, added[0].SubscriptionItem.ActiveFrom)
	}
}

func assertUnchangedMigrationBilling(t *testing.T, fixture migrationFixture, after subscription.SubscriptionView) {
	t.Helper()
	key := subscriptiontestutils.ExampleFeatureKey
	beforeBilling := migrationBillingState(t, fixture.before, fixture.at)
	afterBilling := migrationBillingState(t, after, fixture.at)
	// Unchanged items must keep the same invoice time and billing periods.
	for _, old := range beforeBilling.Items {
		if old.Spec.ItemKey == key && (fixture.scenario == "change price" || fixture.scenario == "remove item") {
			continue
		}
		found, ok := lo.Find(afterBilling.Items, func(item targetstate.StateItem) bool { return item.UniqueID == old.UniqueID })
		require.True(t, ok, "missing billable %s", old.UniqueID)
		require.Equal(t, old.ServicePeriod, found.ServicePeriod)
		require.Equal(t, old.FullServicePeriod, found.FullServicePeriod)
		require.Equal(t, old.BillingPeriod, found.BillingPeriod)
		require.Equal(t, old.GetInvoiceAt(), found.GetInvoiceAt())
	}
}

func assertStalePlanEditRejected(t *testing.T, fixture migrationFixture) {
	t.Helper()
	_, err := fixture.deps.SubscriptionService.Update(t.Context(), fixture.before.Subscription.NamespacedID, fixture.before.Spec)
	require.Error(t, err)
}

func migrationBillingState(t *testing.T, view subscription.SubscriptionView, at time.Time) targetstate.State {
	t.Helper()
	builder := targetstate.NewBuilder(testutils.NewLogger(t), noop.NewTracerProvider().Tracer("test"))
	state, err := builder.Build(t.Context(), targetstate.BuildInput{
		Persisted: persistedstate.State{ByUniqueID: map[string]persistedstate.Item{}, Invoices: persistedstate.Invoices{}},
		AsOf:      at, SubscriptionView: &view,
	})
	require.NoError(t, err)
	return state
}
