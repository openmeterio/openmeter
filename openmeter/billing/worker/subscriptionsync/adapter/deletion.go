package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionsyncstatev2"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func (a *adapter) ListDeletedSubscriptionsToReconcile(ctx context.Context, input subscriptionsync.ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[models.NamespacedID], error) {
	if err := input.Validate(); err != nil {
		return paginationv2.Result[models.NamespacedID]{}, fmt.Errorf("validating deleted subscription list: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (paginationv2.Result[models.NamespacedID], error) {
		query, args := DeletedSubscriptionsToReconcileQuery(tx.db.GetConfig().Driver.Dialect(), input)
		rows, err := tx.db.QueryContext(ctx, query, args...)
		if err != nil {
			return paginationv2.Result[models.NamespacedID]{}, fmt.Errorf("listing deleted subscriptions awaiting reconciliation: %w", err)
		}

		defer func() { _ = rows.Close() }()

		result := paginationv2.Result[models.NamespacedID]{Items: []models.NamespacedID{}}
		var lastCreatedAt time.Time
		for rows.Next() {
			var id models.NamespacedID
			if err := rows.Scan(&id.ID, &id.Namespace, &lastCreatedAt); err != nil {
				return paginationv2.Result[models.NamespacedID]{}, fmt.Errorf("scanning deleted subscription: %w", err)
			}

			result.Items = append(result.Items, id)
		}

		if err := rows.Err(); err != nil {
			return paginationv2.Result[models.NamespacedID]{}, fmt.Errorf("iterating deleted subscriptions: %w", err)
		}

		if len(result.Items) == input.Limit {
			last := result.Items[len(result.Items)-1]
			result.NextCursor = lo.ToPtr(paginationv2.NewCursor(lastCreatedAt, last.ID))
		}

		return result, nil
	})
}

// DeletedSubscriptionsToReconcileQuery builds the cleanup query without executing
// it, so SQL tests can preserve the namespace filters and index-seeking cursor.
func DeletedSubscriptionsToReconcileQuery(dialect string, input subscriptionsync.ListDeletedSubscriptionsToReconcileInput) (string, []any) {
	completed := []predicate.SubscriptionSyncStateV2{
		subscriptionsyncstatev2.LastSuccessStateEQ(subscriptionsync.SyncStateV2StateSubscriptionDeleted),
	}
	if len(input.Namespaces) > 0 {
		// A namespace-scoped worker must not scan every tenant's cleanup records.
		completed = append(completed, subscriptionsyncstatev2.NamespaceIn(input.Namespaces...))
	}

	sub := sql.Table(dbsubscription.Table)
	query := sql.Dialect(dialect).Select(
		sub.C(dbsubscription.FieldID),
		sub.C(dbsubscription.FieldNamespace),
		sub.C(dbsubscription.FieldCreatedAt),
	).From(sub)
	// Deleted subscriptions were scheduled for the future. Their gathering lines
	// need cleanup before billing collects them, even if DeletedAt is ahead of
	// this worker's clock; it is not a scheduled deletion time.
	dbsubscription.DeletedAtNotNil()(query)
	dbsubscription.Not(dbsubscription.HasSyncStateV2With(completed...))(query)
	if len(input.Namespaces) > 0 {
		dbsubscription.NamespaceIn(input.Namespaces...)(query)
	}

	if len(input.Customers) > 0 {
		dbsubscription.CustomerIDIn(input.Customers...)(query)
	}

	if input.Cursor != nil {
		// Match the index's (created_at, id) order so later pages can seek
		// past earlier subscriptions instead of filtering them again.
		query.Where(sql.CompositeGT(
			[]string{sub.C(dbsubscription.FieldCreatedAt), sub.C(dbsubscription.FieldID)},
			input.Cursor.Time, input.Cursor.ID,
		))
	}

	return query.OrderBy(sql.Asc(sub.C(dbsubscription.FieldCreatedAt)), sql.Asc(sub.C(dbsubscription.FieldID))).
		Limit(input.Limit).Query()
}
