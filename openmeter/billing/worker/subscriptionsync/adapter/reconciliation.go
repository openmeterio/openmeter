package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionsyncstatev2"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func (a *adapter) ListSubscriptionsToReconcile(ctx context.Context, input subscriptionsync.ListSubscriptionsToReconcileInput) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
	if err := input.Validate(); err != nil {
		return paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{}, fmt.Errorf("validating subscription list: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
		query, args := SubscriptionsToReconcileQuery(tx.db.GetConfig().Driver.Dialect(), input)
		rows, err := tx.db.QueryContext(ctx, query, args...)
		if err != nil {
			return paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{}, fmt.Errorf("listing subscriptions awaiting reconciliation: %w", err)
		}

		defer func() { _ = rows.Close() }()

		result := paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{Items: []subscriptionsync.SubscriptionToReconcile{}}
		var lastCreatedAt time.Time
		for rows.Next() {
			var item subscriptionsync.SubscriptionToReconcile
			if err := rows.Scan(&item.ID, &item.Namespace, &item.CustomerID, &lastCreatedAt); err != nil {
				return paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{}, fmt.Errorf("scanning subscription: %w", err)
			}

			result.Items = append(result.Items, item)
		}

		if err := rows.Err(); err != nil {
			return paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{}, fmt.Errorf("iterating subscriptions: %w", err)
		}

		if len(result.Items) == input.Limit {
			last := result.Items[len(result.Items)-1]
			result.NextCursor = lo.ToPtr(paginationv2.NewCursor(lastCreatedAt, last.ID))
		}

		return result, nil
	})
}

// SubscriptionsToReconcileQuery builds the SQL used by the adapter and the
// benchmark's EXPLAIN, so both measure the same candidate query.
// Reproduce BenchmarkListSubscriptionsToReconcile in reconciliation_benchmark_test.go
// from the repository root with local PostgreSQL running (200,000 synthetic subscriptions):
//
//	direnv exec . env POSTGRES_HOST=127.0.0.1 POSTGRES_PORT=5432 SUBSCRIPTION_SYNC_BENCH_PLAN_DIR=/tmp/subscription-sync-plans go test -tags=dynamic ./openmeter/billing/worker/subscriptionsync/adapter -run '^$' -bench BenchmarkListSubscriptionsToReconcile -benchtime=20x -count=3
//
// Set POSTGRES_PORT to the local database port. SUBSCRIPTION_SYNC_BENCH_PLAN_DIR
// saves the SQL and EXPLAIN (ANALYZE, BUFFERS) plans for each benchmark scenario.
func SubscriptionsToReconcileQuery(dialect string, input subscriptionsync.ListSubscriptionsToReconcileInput) (string, []any) {
	asOf := input.AsOf.UTC()

	sub := sql.Table(dbsubscription.Table)
	state := sql.Table(subscriptionsyncstatev2.Table)
	query := sql.Dialect(dialect).Select(
		sub.C(dbsubscription.FieldID),
		sub.C(dbsubscription.FieldNamespace),
		sub.C(dbsubscription.FieldCustomerID),
		sub.C(dbsubscription.FieldCreatedAt),
	).From(sub)

	// Deleted subscriptions belong to the cleanup query, even when their
	// DeletedAt is still ahead of the worker clock.
	dbsubscription.DeletedAtIsNil()(query)
	// Active at some point in the lookback.
	dbsubscription.ActiveFromLT(asOf)(query)
	dbsubscription.Or(
		dbsubscription.ActiveToIsNil(),
		dbsubscription.ActiveToGT(asOf.Add(-input.Lookback)),
	)(query)
	if len(input.Namespaces) > 0 {
		dbsubscription.NamespaceIn(input.Namespaces...)(query)
	}

	if len(input.Customers) > 0 {
		dbsubscription.CustomerIDIn(input.Customers...)(query)
	}

	if !input.ForceSync {
		// A subscription is due unless it has a state row that needs nothing, which
		// also covers a subscription with no row at all. This query is the only
		// place the rule lives; the README's periodic reconciliation table
		// describes it.
		//
		// Do not turn this into a LEFT JOIN filtered on the due conditions.
		// PostgreSQL assumes that nearly every row has differing revisions, expects
		// to fill the page from the first few subscriptions, and probes the state
		// of every subscription one by one: five times slower in the benchmark.
		settled := sql.Dialect(dialect).Select(state.C(subscriptionsyncstatev2.FieldID)).From(state).Where(sql.And(
			sql.ColumnsEQ(state.C(subscriptionsyncstatev2.FieldSubscriptionID), sub.C(dbsubscription.FieldID)),
			// Subscription IDs are globally unique, but a state row must also match
			// the namespace. One from another namespace counts as missing.
			sql.ColumnsEQ(state.C(subscriptionsyncstatev2.FieldNamespace), sub.C(dbsubscription.FieldNamespace)),
			// A sync has recorded an outcome. A new row starts with equal revisions,
			// so the revision check alone would treat it as settled.
			sql.NEQ(state.C(subscriptionsyncstatev2.FieldLastSuccessState), subscriptionsync.SyncStateV2StateNew),
			// The subscription has not changed since. A change also reopens
			// outcomes such as ended or no_billables.
			sql.ColumnsEQ(
				state.C(subscriptionsyncstatev2.FieldCurrentSubsRevision),
				state.C(subscriptionsyncstatev2.FieldLastSyncedSubsRevision),
			),
			// An active subscription has not reached its deadline.
			sql.Not(sql.And(
				sql.EQ(state.C(subscriptionsyncstatev2.FieldLastSuccessState), subscriptionsync.SyncStateV2StateActive),
				sql.LTE(state.C(subscriptionsyncstatev2.FieldNextReconciliationAt), asOf),
			)),
		))
		query.Where(sql.NotExists(settled))
	}

	if input.Cursor != nil {
		query.Where(sql.CompositeGT(
			[]string{sub.C(dbsubscription.FieldCreatedAt), sub.C(dbsubscription.FieldID)},
			input.Cursor.Time, input.Cursor.ID,
		))
	}

	return query.OrderBy(sql.Asc(sub.C(dbsubscription.FieldCreatedAt)), sql.Asc(sub.C(dbsubscription.FieldID))).
		Limit(input.Limit).Query()
}
