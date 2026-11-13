package adapter_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/oklog/ulid/v2"
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

type DeletedSubscriptionsToReconcileSuite struct {
	suite.Suite
	testDB     *testutils.TestDB
	db         *entdb.Client
	adapter    subscriptionsync.Adapter
	now        time.Time
	namespace  string
	customerID models.NamespacedID
}

func TestListDeletedSubscriptionsToReconcile(t *testing.T) {
	suite.Run(t, new(DeletedSubscriptionsToReconcileSuite))
}

func (s *DeletedSubscriptionsToReconcileSuite) SetupTest() {
	t := s.T()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	s.testDB = testDB
	s.db = testDB.EntDriver.Client()
	adapter, err := syncadapter.New(syncadapter.Config{Client: s.db})
	s.Require().NoError(err)
	s.adapter = adapter
	s.now = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	s.namespace = ulid.Make().String()
	customer, err := s.db.Customer.Create().SetNamespace(s.namespace).SetName("Cleanup customer").Save(t.Context())
	s.Require().NoError(err)
	s.customerID = models.NamespacedID{Namespace: s.namespace, ID: customer.ID}
}

func (s *DeletedSubscriptionsToReconcileSuite) createSubscription(customerID models.NamespacedID) subscription.Subscription {
	s.T().Helper()

	sub, err := subscriptionrepo.NewSubscriptionRepo(s.db).Create(s.T().Context(), subscription.CreateSubscriptionEntityInput{
		NamespacedModel: models.NamespacedModel{Namespace: customerID.Namespace},
		CadencedModel:   models.CadencedModel{ActiveFrom: s.now.AddDate(-2, 0, 0)},
		Name:            "Cleanup subscription",
		CustomerId:      customerID.ID,
		InvoiceCurrency: "USD",
		BillingCadence:  datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
		BillingAnchor:   s.now,
		SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
	})
	s.Require().NoError(err)

	return sub
}

func (s *DeletedSubscriptionsToReconcileSuite) recordDeletionCleanup(subscriptionID models.NamespacedID) {
	s.T().Helper()

	s.Require().NoError(s.adapter.UpsertSyncStateV2(s.T().Context(), subscriptionsync.UpsertSyncStateV2Input{
		SubscriptionID:   subscriptionID,
		LastSuccessState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		LastSuccessAt:    s.now,
	}))
}

func (s *DeletedSubscriptionsToReconcileSuite) TestDeletionTimeCutoff() {
	ctx := s.T().Context()

	// given: live subscriptions and deletions before, at, and after the cutoff
	s.createSubscription(s.customerID)
	past := s.createSubscription(s.customerID)
	atCutoff := s.createSubscription(s.customerID)
	future := s.createSubscription(s.customerID)
	s.Require().NoError(s.db.Subscription.UpdateOneID(past.ID).SetDeletedAt(s.now.Add(-time.Hour)).Exec(ctx))
	s.Require().NoError(s.db.Subscription.UpdateOneID(atCutoff.ID).SetDeletedAt(s.now).Exec(ctx))
	s.Require().NoError(s.db.Subscription.UpdateOneID(future.ID).SetDeletedAt(s.now.Add(time.Hour)).Exec(ctx))

	// when: the worker selects subscriptions deleted by the cutoff
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 100,
	})

	// then: live subscriptions and future deletions are excluded
	s.Require().NoError(err)
	s.ElementsMatch([]models.NamespacedID{past.NamespacedID, atCutoff.NamespacedID}, result.Items)
	s.Nil(result.NextCursor, "a partial page has no continuation")
}

func (s *DeletedSubscriptionsToReconcileSuite) TestHistoricalDeletionsIgnoreServicePeriod() {
	ctx := s.T().Context()

	// given: old deletions with and without a service end
	old := s.now.AddDate(-1, 0, 0)
	withoutEnd := s.createSubscription(s.customerID)
	withEnd := s.createSubscription(s.customerID)
	s.Require().NoError(s.db.Subscription.UpdateOneID(withoutEnd.ID).SetDeletedAt(old).Exec(ctx))
	s.Require().NoError(s.db.Subscription.UpdateOneID(withEnd.ID).SetDeletedAt(old).SetActiveTo(old).Exec(ctx))

	// when: cleanup candidates are listed long after deletion
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 100,
	})

	// then: neither the deletion time nor the service end imposes a lookback cutoff
	s.Require().NoError(err)
	s.ElementsMatch([]models.NamespacedID{withoutEnd.NamespacedID, withEnd.NamespacedID}, result.Items)
}

func (s *DeletedSubscriptionsToReconcileSuite) TestLegacyStateDoesNotCompleteCleanup() {
	ctx := s.T().Context()

	// given: a deleted subscription previously marked as having no billables
	sub := s.createSubscription(s.customerID)
	s.Require().NoError(s.db.Subscription.UpdateOneID(sub.ID).SetDeletedAt(s.now).Exec(ctx))
	s.Require().NoError(s.adapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
		SubscriptionID: sub.NamespacedID,
		SyncedAt:       s.now,
		HasBillables:   false,
	}))

	// when: the worker selects cleanup candidates
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 100,
	})

	// then: legacy state cannot stand in for a v2 deletion cleanup result
	s.Require().NoError(err)
	s.Equal([]models.NamespacedID{sub.NamespacedID}, result.Items)
}

func (s *DeletedSubscriptionsToReconcileSuite) TestCompletedDeletionIsExcluded() {
	ctx := s.T().Context()

	// given: a deleted subscription whose cleanup completed successfully
	sub := s.createSubscription(s.customerID)
	s.Require().NoError(s.db.Subscription.UpdateOneID(sub.ID).SetDeletedAt(s.now).Exec(ctx))
	s.recordDeletionCleanup(sub.NamespacedID)

	// when: the worker selects cleanup candidates
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 100,
	})

	// then: completed cleanup is not scheduled again
	s.Require().NoError(err)
	s.Empty(result.Items)
	s.Nil(result.NextCursor)
}

func (s *DeletedSubscriptionsToReconcileSuite) TestDifferentV2ResultStillRequiresCleanup() {
	ctx := s.T().Context()

	// given: a deleted subscription with a v2 result for work other than deletion
	sub := s.createSubscription(s.customerID)
	s.Require().NoError(s.db.Subscription.UpdateOneID(sub.ID).SetDeletedAt(s.now).Exec(ctx))
	s.recordDeletionCleanup(sub.NamespacedID)
	// The Go enum currently supports only deletion. Seed a future state directly
	// to verify that the query matches the result, not merely row existence.
	_, err := s.testDB.PGDriver.DB().ExecContext(ctx,
		"UPDATE subscription_sync_states_v2 SET last_success_state = $1 WHERE subscription_id = $2", "active", sub.ID)
	s.Require().NoError(err)

	// when: the worker selects cleanup candidates
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 100,
	})

	// then: that other result does not establish deletion cleanup
	s.Require().NoError(err)
	s.Equal([]models.NamespacedID{sub.NamespacedID}, result.Items)
}

func (s *DeletedSubscriptionsToReconcileSuite) TestNamespaceAndCustomerFilters() {
	ctx := s.T().Context()

	// given: deleted subscriptions for two customers in one namespace and a third in another
	otherCustomer, err := s.db.Customer.Create().SetNamespace(s.namespace).SetName("Other customer").Save(ctx)
	s.Require().NoError(err)
	otherNamespace := ulid.Make().String()
	foreignCustomer, err := s.db.Customer.Create().SetNamespace(otherNamespace).SetName("Other namespace").Save(ctx)
	s.Require().NoError(err)
	first := s.createSubscription(s.customerID)
	second := s.createSubscription(models.NamespacedID{Namespace: s.namespace, ID: otherCustomer.ID})
	foreign := s.createSubscription(models.NamespacedID{Namespace: otherNamespace, ID: foreignCustomer.ID})
	for _, sub := range []subscription.Subscription{first, second, foreign} {
		s.Require().NoError(s.db.Subscription.UpdateOneID(sub.ID).SetDeletedAt(s.now).Exec(ctx))
	}

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
			result, err := s.adapter.ListDeletedSubscriptionsToReconcile(s.T().Context(), subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
				Namespaces: tc.namespaces,
				Customers:  tc.customers,
				AsOf:       s.now,
				Limit:      100,
			})

			// then: every returned subscription belongs to the requested scope
			s.Require().NoError(err)
			s.ElementsMatch(tc.want, result.Items)
		})
	}
}

func (s *DeletedSubscriptionsToReconcileSuite) TestPaginationWhileCleanupCompletes() {
	ctx := s.T().Context()

	// given: deleted subscriptions with both distinct and tied creation times
	clock.FreezeTime(s.now)
	defer clock.UnFreeze()
	later := s.createSubscription(s.customerID)
	tied := s.createSubscription(s.customerID)
	clock.FreezeTime(s.now.Add(-time.Hour))
	defer clock.UnFreeze()
	earlier := s.createSubscription(s.customerID)
	for _, sub := range []subscription.Subscription{later, tied, earlier} {
		s.Require().NoError(s.db.Subscription.UpdateOneID(sub.ID).SetDeletedAt(s.now).Exec(ctx))
	}

	laterIDs := []models.NamespacedID{later.NamespacedID, tied.NamespacedID}
	slices.SortFunc(laterIDs, func(a, b models.NamespacedID) int { return strings.Compare(a.ID, b.ID) })
	expected := append([]models.NamespacedID{earlier.NamespacedID}, laterIDs...)
	input := subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		AsOf:  s.now,
		Limit: 1,
	}

	// when: each returned subscription finishes cleanup before the next page is read
	var visited []models.NamespacedID
	for range expected {
		page, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, input)
		s.Require().NoError(err)
		s.Require().Len(page.Items, 1)
		visited = append(visited, page.Items[0])
		s.recordDeletionCleanup(page.Items[0])
		s.Require().NotNil(page.NextCursor)
		input.Cursor = page.NextCursor
	}

	// then: the cursor visits every candidate exactly once in creation-time and ID order
	s.Equal(expected, visited)
	result, err := s.adapter.ListDeletedSubscriptionsToReconcile(ctx, input)
	s.Require().NoError(err)
	s.Empty(result.Items)
	s.Nil(result.NextCursor)
	input.Cursor = nil
	result, err = s.adapter.ListDeletedSubscriptionsToReconcile(ctx, input)
	s.Require().NoError(err)
	s.Empty(result.Items, "completed cleanup must stay excluded when polling starts over")
}

func TestDeletedSubscriptionsToReconcileQuery(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	cursor := paginationv2.NewCursor(now.Add(-time.Hour), "00000000000000000000000100")
	tests := []struct {
		name          string
		input         subscriptionsync.ListDeletedSubscriptionsToReconcileInput
		expectedQuery string
		expectedArgs  []any
	}{
		{
			name: "global first page",
			input: subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
				AsOf:  now,
				Limit: 100,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."created_at" FROM "subscriptions" WHERE "subscriptions"."deleted_at" <= $1 AND (NOT (EXISTS (SELECT "subscription_sync_states_v2"."subscription_id" FROM "subscription_sync_states_v2" WHERE "subscriptions"."id" = "subscription_sync_states_v2"."subscription_id" AND "subscription_sync_states_v2"."last_success_state" = $2))) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 100`,
			expectedArgs:  []any{now, subscriptionsync.SyncStateV2StateSubscriptionDeleted},
		},
		{
			name: "namespace first page",
			input: subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
				AsOf:       now,
				Namespaces: []string{"namespace-1"},
				Limit:      100,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."created_at" FROM "subscriptions" WHERE ("subscriptions"."deleted_at" <= $1 AND (NOT (EXISTS (SELECT "subscription_sync_states_v2"."subscription_id" FROM "subscription_sync_states_v2" WHERE ("subscriptions"."id" = "subscription_sync_states_v2"."subscription_id" AND "subscription_sync_states_v2"."last_success_state" = $2) AND "subscription_sync_states_v2"."namespace" IN ($3))))) AND "subscriptions"."namespace" IN ($4) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 100`,
			expectedArgs:  []any{now, subscriptionsync.SyncStateV2StateSubscriptionDeleted, "namespace-1", "namespace-1"},
		},
		{
			name: "customer cursor page in two namespaces",
			input: subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
				AsOf:       now,
				Namespaces: []string{"namespace-1", "namespace-2"},
				Customers:  []string{"customer-1"},
				Cursor:     &cursor,
				Limit:      10,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."created_at" FROM "subscriptions" WHERE ((("subscriptions"."deleted_at" <= $1 AND (NOT (EXISTS (SELECT "subscription_sync_states_v2"."subscription_id" FROM "subscription_sync_states_v2" WHERE ("subscriptions"."id" = "subscription_sync_states_v2"."subscription_id" AND "subscription_sync_states_v2"."last_success_state" = $2) AND "subscription_sync_states_v2"."namespace" IN ($3, $4))))) AND "subscriptions"."namespace" IN ($5, $6)) AND "subscriptions"."customer_id" IN ($7)) AND ("subscriptions"."created_at", "subscriptions"."id") > ($8, $9) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 10`,
			expectedArgs:  []any{now, subscriptionsync.SyncStateV2StateSubscriptionDeleted, "namespace-1", "namespace-2", "namespace-1", "namespace-2", "customer-1", cursor.Time, cursor.ID},
		},
		{
			name: "global cursor page",
			input: subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
				AsOf:   now,
				Cursor: &cursor,
				Limit:  10,
			},
			expectedQuery: `SELECT "subscriptions"."id", "subscriptions"."namespace", "subscriptions"."created_at" FROM "subscriptions" WHERE ("subscriptions"."deleted_at" <= $1 AND (NOT (EXISTS (SELECT "subscription_sync_states_v2"."subscription_id" FROM "subscription_sync_states_v2" WHERE "subscriptions"."id" = "subscription_sync_states_v2"."subscription_id" AND "subscription_sync_states_v2"."last_success_state" = $2)))) AND ("subscriptions"."created_at", "subscriptions"."id") > ($3, $4) ORDER BY "subscriptions"."created_at" ASC, "subscriptions"."id" ASC LIMIT 10`,
			expectedArgs:  []any{now, subscriptionsync.SyncStateV2StateSubscriptionDeleted, cursor.Time, cursor.ID},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, args := syncadapter.DeletedSubscriptionsToReconcileQuery(dialect.Postgres, tc.input)
			require.Equal(t, tc.expectedQuery, query)
			require.Equal(t, tc.expectedArgs, args)
		})
	}
}
