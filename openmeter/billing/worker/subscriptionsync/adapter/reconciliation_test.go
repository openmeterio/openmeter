package adapter_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const reconciliationLookback = 24 * time.Hour

type SubscriptionsToReconcileSuite struct {
	suite.Suite
	db         *entdb.Client
	adapter    subscriptionsync.Adapter
	now        time.Time
	namespace  string
	customerID models.NamespacedID
}

func TestListSubscriptionsToReconcile(t *testing.T) {
	suite.Run(t, new(SubscriptionsToReconcileSuite))
}

func (s *SubscriptionsToReconcileSuite) SetupTest() {
	t := s.T()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	s.db = testDB.EntDriver.Client()
	adapter, err := syncadapter.New(syncadapter.Config{Client: s.db})
	s.Require().NoError(err)
	s.adapter = adapter
	s.now = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	s.namespace = ulid.Make().String()
	customer, err := s.db.Customer.Create().SetNamespace(s.namespace).SetName("Reconciliation customer").Save(t.Context())
	s.Require().NoError(err)
	s.customerID = models.NamespacedID{Namespace: s.namespace, ID: customer.ID}
}

// createSubscription stores a subscription active over the given period; a nil
// activeTo leaves it running.
func (s *SubscriptionsToReconcileSuite) createSubscription(customerID models.NamespacedID, activeFrom time.Time, activeTo *time.Time) subscription.Subscription {
	s.T().Helper()

	sub, err := subscriptionrepo.NewSubscriptionRepo(s.db).Create(s.T().Context(), subscription.CreateSubscriptionEntityInput{
		NamespacedModel: models.NamespacedModel{Namespace: customerID.Namespace},
		CadencedModel:   models.CadencedModel{ActiveFrom: activeFrom, ActiveTo: activeTo},
		Name:            "Reconciliation subscription",
		CustomerId:      customerID.ID,
		InvoiceCurrency: "USD",
		BillingCadence:  datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
		BillingAnchor:   s.now,
		SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
	})
	s.Require().NoError(err)

	return sub
}

func (s *SubscriptionsToReconcileSuite) createRunningSubscription() subscription.Subscription {
	s.T().Helper()

	return s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), nil)
}

// recordSettled records an active result whose deadline is still a day ahead.
func (s *SubscriptionsToReconcileSuite) recordSettled(subscriptionID models.NamespacedID) {
	s.T().Helper()

	s.Require().NoError(s.adapter.RecordSyncResult(s.T().Context(), subscriptionsync.RecordSyncResultInput{
		SubscriptionID:         subscriptionID,
		LastSuccessState:       subscriptionsync.SyncStateStateActive,
		LastSuccessAt:          s.now.Add(-time.Hour),
		NextReconciliationAt:   lo.ToPtr(s.now.Add(24 * time.Hour)),
		LastSyncedSubsRevision: ulid.Make().String(),
	}))
}

// subscriptionIDs drops the customer from the listed candidates, so they can be
// compared against the subscriptions a test created.
func subscriptionIDs(items []subscriptionsync.SubscriptionToReconcile) []models.NamespacedID {
	return lo.Map(items, func(item subscriptionsync.SubscriptionToReconcile, _ int) models.NamespacedID {
		return item.NamespacedID
	})
}

func (s *SubscriptionsToReconcileSuite) listInput() subscriptionsync.ListSubscriptionsToReconcileInput {
	return subscriptionsync.ListSubscriptionsToReconcileInput{
		Namespaces: []string{s.namespace},
		AsOf:       s.now,
		Lookback:   reconciliationLookback,
		Limit:      100,
	}
}

func (s *SubscriptionsToReconcileSuite) TestSyncStateDecidesWhatIsDue() {
	ctx := s.T().Context()

	tests := []struct {
		name string
		// result is the recorded sync outcome; nil leaves the subscription without one.
		result *subscriptionsync.RecordSyncResultInput
		// changed marks a subscription change after the outcome, or creates a new row when there is none.
		changed bool
		due     bool
	}{
		{
			name: "no state row",
			due:  true,
		},
		{
			name:    "new row without an outcome",
			changed: true,
			due:     true,
		},
		{
			name:   "active before its deadline",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateActive, NextReconciliationAt: lo.ToPtr(s.now.Add(time.Microsecond))},
		},
		{
			name:   "active at its deadline",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateActive, NextReconciliationAt: lo.ToPtr(s.now)},
			due:    true,
		},
		{
			name:   "active past its deadline",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateActive, NextReconciliationAt: lo.ToPtr(s.now.Add(-time.Hour))},
			due:    true,
		},
		{
			name:    "active and changed before its deadline",
			result:  &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateActive, NextReconciliationAt: lo.ToPtr(s.now.Add(time.Hour))},
			changed: true,
			due:     true,
		},
		{
			name:   "no billables",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateNoBillables},
		},
		{
			name:    "no billables then changed",
			result:  &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateNoBillables},
			changed: true,
			due:     true,
		},
		{
			name:   "ended",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateEnded},
		},
		{
			name:    "ended then changed",
			result:  &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateEnded},
			changed: true,
			due:     true,
		},
		{
			name:   "too old",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateTooOld},
		},
		{
			name:   "customer deleted before start",
			result: &subscriptionsync.RecordSyncResultInput{LastSuccessState: subscriptionsync.SyncStateStateCustomerDeletedBefore},
		},
	}

	// given: one running subscription per sync state
	var wantDue []models.NamespacedID
	for _, tc := range tests {
		sub := s.createRunningSubscription()
		if tc.result != nil {
			result := *tc.result
			result.SubscriptionID = sub.NamespacedID
			result.LastSuccessAt = s.now.Add(-time.Hour)
			result.LastSyncedSubsRevision = ulid.Make().String()
			s.Require().NoError(s.adapter.RecordSyncResult(ctx, result), tc.name)
		}

		if tc.changed {
			s.Require().NoError(s.adapter.MarkSubscriptionChanged(ctx, sub.NamespacedID), tc.name)
		}

		if tc.due {
			wantDue = append(wantDue, sub.NamespacedID)
		}
	}

	// when: the worker selects subscriptions to reconcile
	result, err := s.adapter.ListSubscriptionsToReconcile(ctx, s.listInput())

	// then: exactly the due subscriptions are selected
	s.Require().NoError(err)
	s.ElementsMatch(wantDue, subscriptionIDs(result.Items))
	s.Nil(result.NextCursor, "a partial page has no continuation")
}

func (s *SubscriptionsToReconcileSuite) TestLookbackAndDeletion() {
	ctx := s.T().Context()
	lookbackStart := s.now.Add(-reconciliationLookback)

	// given: subscriptions without sync state around the edges of the lookback
	running := s.createRunningSubscription()
	startedInLookback := s.createSubscription(s.customerID, s.now.Add(-time.Hour), nil)
	endedInLookback := s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), lo.ToPtr(lookbackStart.Add(time.Microsecond)))
	endsLater := s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), lo.ToPtr(s.now.AddDate(0, 1, 0)))
	s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), lo.ToPtr(lookbackStart))
	s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), lo.ToPtr(s.now.AddDate(0, -1, 0)))
	s.createSubscription(s.customerID, s.now, nil)
	s.createSubscription(s.customerID, s.now.Add(time.Hour), nil)
	// Deleted subscriptions belong to the cleanup query, whatever their deletion timestamp.
	for _, deletedAt := range []time.Time{s.now.Add(-time.Hour), s.now.Add(time.Hour)} {
		deleted := s.createRunningSubscription()
		s.Require().NoError(s.db.Subscription.UpdateOneID(deleted.ID).SetDeletedAt(deletedAt).Exec(ctx))
	}

	// when: the worker selects subscriptions to reconcile
	result, err := s.adapter.ListSubscriptionsToReconcile(ctx, s.listInput())

	// then: only non-deleted subscriptions active at some point within the lookback are selected
	s.Require().NoError(err)
	s.ElementsMatch([]models.NamespacedID{
		running.NamespacedID,
		startedInLookback.NamespacedID,
		endedInLookback.NamespacedID,
		endsLater.NamespacedID,
	}, subscriptionIDs(result.Items))

	// and: a wider lookback reaches the subscription that ended a month ago
	input := s.listInput()
	input.Lookback = 60 * 24 * time.Hour
	result, err = s.adapter.ListSubscriptionsToReconcile(ctx, input)
	s.Require().NoError(err)
	s.Len(result.Items, 6)
}

func (s *SubscriptionsToReconcileSuite) TestForceSync() {
	ctx := s.T().Context()

	// given: a settled subscription, one that is due, a deleted one and one outside the lookback
	settled := s.createRunningSubscription()
	s.recordSettled(settled.NamespacedID)
	due := s.createRunningSubscription()
	deleted := s.createRunningSubscription()
	s.Require().NoError(s.db.Subscription.UpdateOneID(deleted.ID).SetDeletedAt(s.now).Exec(ctx))
	s.createSubscription(s.customerID, s.now.AddDate(-1, 0, 0), lo.ToPtr(s.now.AddDate(0, -1, 0)))

	// when: forced reconciliation asks for everything in the lookback
	input := s.listInput()
	input.ForceSync = true
	result, err := s.adapter.ListSubscriptionsToReconcile(ctx, input)

	// then: sync state is ignored, but deletion and the lookback still apply
	s.Require().NoError(err)
	s.ElementsMatch([]models.NamespacedID{settled.NamespacedID, due.NamespacedID}, subscriptionIDs(result.Items))
}

func (s *SubscriptionsToReconcileSuite) TestStateFromAnotherNamespaceCountsAsMissing() {
	ctx := s.T().Context()

	// given: a settled state row that carries another namespace than its subscription
	sub := s.createRunningSubscription()
	revision := ulid.Make().String()
	s.Require().NoError(s.db.SubscriptionSyncState.Create().
		SetNamespace(ulid.Make().String()).
		SetSubscriptionID(sub.ID).
		SetLastSuccessState(subscriptionsync.SyncStateStateNoBillables).
		SetLastSuccessAt(s.now).
		SetLastSyncedSubsRevision(revision).
		SetCurrentSubsRevision(revision).
		Exec(ctx))

	// when: the worker selects subscriptions to reconcile
	result, err := s.adapter.ListSubscriptionsToReconcile(ctx, s.listInput())

	// then: the foreign row cannot settle the subscription, matching GetSyncStates
	s.Require().NoError(err)
	s.Equal([]subscriptionsync.SubscriptionToReconcile{{NamespacedID: sub.NamespacedID, CustomerID: s.customerID.ID}}, result.Items)
	states, err := s.adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{sub.NamespacedID})
	s.Require().NoError(err)
	s.Empty(states)
}

func (s *SubscriptionsToReconcileSuite) TestNamespaceAndCustomerFilters() {
	ctx := s.T().Context()

	// given: due subscriptions for two customers in one namespace and a third in another
	otherCustomer, err := s.db.Customer.Create().SetNamespace(s.namespace).SetName("Other customer").Save(ctx)
	s.Require().NoError(err)
	otherNamespace := ulid.Make().String()
	foreignCustomer, err := s.db.Customer.Create().SetNamespace(otherNamespace).SetName("Other namespace").Save(ctx)
	s.Require().NoError(err)
	activeFrom := s.now.AddDate(-1, 0, 0)
	first := s.createSubscription(s.customerID, activeFrom, nil)
	second := s.createSubscription(models.NamespacedID{Namespace: s.namespace, ID: otherCustomer.ID}, activeFrom, nil)
	foreign := s.createSubscription(models.NamespacedID{Namespace: otherNamespace, ID: foreignCustomer.ID}, activeFrom, nil)

	tests := []struct {
		name       string
		namespaces []string
		customers  []string
		want       []models.NamespacedID
	}{
		{
			name: "all namespaces",
			want: []models.NamespacedID{first.NamespacedID, second.NamespacedID, foreign.NamespacedID},
		},
		{
			name:       "one namespace",
			namespaces: []string{s.namespace},
			want:       []models.NamespacedID{first.NamespacedID, second.NamespacedID},
		},
		{
			name:      "one customer",
			customers: []string{s.customerID.ID},
			want:      []models.NamespacedID{first.NamespacedID},
		},
		{
			name:       "namespace and customer",
			namespaces: []string{s.namespace},
			customers:  []string{s.customerID.ID},
			want:       []models.NamespacedID{first.NamespacedID},
		},
		{
			name:       "customer cannot bypass namespace",
			namespaces: []string{otherNamespace},
			customers:  []string{s.customerID.ID},
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			// when: the worker applies the requested namespace and customer filters
			input := s.listInput()
			input.Namespaces = tc.namespaces
			input.Customers = tc.customers
			result, err := s.adapter.ListSubscriptionsToReconcile(s.T().Context(), input)

			// then: every returned subscription belongs to the requested scope
			s.Require().NoError(err)
			s.ElementsMatch(tc.want, subscriptionIDs(result.Items))
		})
	}
}

func (s *SubscriptionsToReconcileSuite) TestPaginationWhileSyncsComplete() {
	ctx := s.T().Context()

	// given: due subscriptions with both distinct and tied creation times
	clock.FreezeTime(s.now)
	defer clock.UnFreeze()
	later := s.createRunningSubscription()
	tied := s.createRunningSubscription()
	clock.FreezeTime(s.now.Add(-time.Hour))
	defer clock.UnFreeze()
	earlier := s.createRunningSubscription()

	laterIDs := []models.NamespacedID{later.NamespacedID, tied.NamespacedID}
	slices.SortFunc(laterIDs, func(a, b models.NamespacedID) int { return strings.Compare(a.ID, b.ID) })
	expected := append([]models.NamespacedID{earlier.NamespacedID}, laterIDs...)
	input := s.listInput()
	input.Limit = 1

	// when: only the first returned subscription is settled before the next page is
	// read, so the cursor alone has to move past the two that stay listed
	var visited []models.NamespacedID
	for i := range expected {
		page, err := s.adapter.ListSubscriptionsToReconcile(ctx, input)
		s.Require().NoError(err)
		s.Require().Len(page.Items, 1)
		visited = append(visited, page.Items[0].NamespacedID)
		if i == 0 {
			s.recordSettled(page.Items[0].NamespacedID)
		}

		s.Require().NotNil(page.NextCursor)
		input.Cursor = page.NextCursor
	}

	// then: the cursor visits every candidate exactly once in creation-time and ID order
	s.Equal(expected, visited)
	result, err := s.adapter.ListSubscriptionsToReconcile(ctx, input)
	s.Require().NoError(err)
	s.Empty(result.Items)
	s.Nil(result.NextCursor)

	// and: a listing that starts over returns only what is still pending
	input.Cursor = nil
	input.Limit = 100
	result, err = s.adapter.ListSubscriptionsToReconcile(ctx, input)
	s.Require().NoError(err)
	s.Equal(expected[1:], subscriptionIDs(result.Items), "the settled subscription must stay excluded when polling starts over")
}

func TestSubscriptionsToReconcileQuery(t *testing.T) {
	// Result assertions cannot detect a query that returns the same rows through a
	// plan PostgreSQL executes far slower, such as a LEFT JOIN filtered on the due
	// conditions instead of the NOT EXISTS anti-join. Keep exact SQL assertions so
	// builder changes and Ent upgrades require reviewing the query; the benchmark
	// uses this same builder.
	asOf := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	lookbackStart := asOf.Add(-reconciliationLookback)
	cursor := paginationv2.NewCursor(asOf.Add(-time.Hour), "00000000000000000000000100")
	tests := []struct {
		name          string
		input         subscriptionsync.ListSubscriptionsToReconcileInput
		expectedQuery string
		expectedArgs  []any
	}{
		{
			name: "global first page",
			input: subscriptionsync.ListSubscriptionsToReconcileInput{
				AsOf:     asOf,
				Lookback: reconciliationLookback,
				Limit:    100,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."customer_id", "subscriptions"."created_at" FROM "subscriptions" WHERE (("subscriptions"."deleted_at" IS NULL AND "subscriptions"."active_from" < $1) AND ("subscriptions"."active_to" IS NULL OR "subscriptions"."active_to" > $2)) AND NOT EXISTS (SELECT "subscription_sync_states_v2"."id" FROM "subscription_sync_states_v2" WHERE "subscription_sync_states_v2"."subscription_id" = "subscriptions"."id" AND "subscription_sync_states_v2"."namespace" = "subscriptions"."namespace" AND "subscription_sync_states_v2"."last_success_state" <> $3 AND "subscription_sync_states_v2"."current_subs_revision" = "subscription_sync_states_v2"."last_synced_subs_revision" AND (NOT ("subscription_sync_states_v2"."last_success_state" = $4 AND "subscription_sync_states_v2"."next_reconciliation_at" <= $5))) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 100`,
			expectedArgs:  []any{asOf, lookbackStart, subscriptionsync.SyncStateStateNew, subscriptionsync.SyncStateStateActive, asOf},
		},
		{
			name: "customer cursor page in two namespaces",
			input: subscriptionsync.ListSubscriptionsToReconcileInput{
				Namespaces: []string{"namespace-1", "namespace-2"},
				Customers:  []string{"customer-1"},
				AsOf:       asOf,
				Lookback:   reconciliationLookback,
				Cursor:     &cursor,
				Limit:      10,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."customer_id", "subscriptions"."created_at" FROM "subscriptions" WHERE ((((("subscriptions"."deleted_at" IS NULL AND "subscriptions"."active_from" < $1) AND ("subscriptions"."active_to" IS NULL OR "subscriptions"."active_to" > $2)) AND "subscriptions"."namespace" IN ($3, $4)) AND "subscriptions"."customer_id" IN ($5)) AND NOT EXISTS (SELECT "subscription_sync_states_v2"."id" FROM "subscription_sync_states_v2" WHERE "subscription_sync_states_v2"."subscription_id" = "subscriptions"."id" AND "subscription_sync_states_v2"."namespace" = "subscriptions"."namespace" AND "subscription_sync_states_v2"."last_success_state" <> $6 AND "subscription_sync_states_v2"."current_subs_revision" = "subscription_sync_states_v2"."last_synced_subs_revision" AND (NOT ("subscription_sync_states_v2"."last_success_state" = $7 AND "subscription_sync_states_v2"."next_reconciliation_at" <= $8)))) AND ("subscriptions"."created_at", "subscriptions"."id") > ($9, $10) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 10`,
			expectedArgs:  []any{asOf, lookbackStart, "namespace-1", "namespace-2", "customer-1", subscriptionsync.SyncStateStateNew, subscriptionsync.SyncStateStateActive, asOf, cursor.Time, cursor.ID},
		},
		{
			name: "forced namespace page",
			input: subscriptionsync.ListSubscriptionsToReconcileInput{
				Namespaces: []string{"namespace-1"},
				AsOf:       asOf,
				Lookback:   reconciliationLookback,
				ForceSync:  true,
				Limit:      100,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."customer_id", "subscriptions"."created_at" FROM "subscriptions" WHERE (("subscriptions"."deleted_at" IS NULL AND "subscriptions"."active_from" < $1) AND ("subscriptions"."active_to" IS NULL OR "subscriptions"."active_to" > $2)) AND "subscriptions"."namespace" IN ($3) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 100`,
			expectedArgs:  []any{asOf, lookbackStart, "namespace-1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, args := syncadapter.SubscriptionsToReconcileQuery(dialect.Postgres, tc.input)
			require.Equal(t, tc.expectedQuery, query)
			require.Equal(t, tc.expectedArgs, args)
		})
	}
}
