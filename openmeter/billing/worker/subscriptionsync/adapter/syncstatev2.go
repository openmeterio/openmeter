package adapter

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionsyncstatev2"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ subscriptionsync.SyncStateV2Adapter = (*adapter)(nil)

func (a *adapter) GetSyncStatesV2(ctx context.Context, input subscriptionsync.GetSyncStatesV2Input) ([]subscriptionsync.SyncStateV2Managed, error) {
	if len(input) == 0 {
		return []subscriptionsync.SyncStateV2Managed{}, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]subscriptionsync.SyncStateV2Managed, error) {
		requestedIDs := lo.SliceToMap(input, func(id models.NamespacedID) (models.NamespacedID, struct{}) {
			return id, struct{}{}
		})
		ids := lo.Map(input, func(id models.NamespacedID, _ int) string { return id.ID })
		states, err := tx.db.SubscriptionSyncStateV2.Query().Where(subscriptionsyncstatev2.SubscriptionIDIn(ids...)).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting subscription sync states v2: %w", err)
		}

		// Subscription IDs are globally unique, but callers must also match the namespace.
		return lo.FilterMap(states, func(state *entdb.SubscriptionSyncStateV2, _ int) (subscriptionsync.SyncStateV2Managed, bool) {
			mappedState := mapSyncStateV2FromDB(state)
			_, requested := requestedIDs[mappedState.SubscriptionID]

			return mappedState, requested
		}), nil
	})
}

func mapSyncStateV2FromDB(state *entdb.SubscriptionSyncStateV2) subscriptionsync.SyncStateV2Managed {
	return subscriptionsync.SyncStateV2Managed{
		SyncStateV2: subscriptionsync.SyncStateV2{
			SubscriptionID: models.NamespacedID{
				Namespace: state.Namespace,
				ID:        state.SubscriptionID,
			},
			LastSuccessState:       state.LastSuccessState,
			LastSuccessAt:          convert.SafeToUTC(state.LastSuccessAt),
			NextReconciliationAt:   convert.SafeToUTC(state.NextReconciliationAt),
			LastSyncedSubsRevision: state.LastSyncedSubsRevision,
			CurrentSubsRevision:    state.CurrentSubsRevision,
		},
		NamespacedID: models.NamespacedID{
			Namespace: state.Namespace,
			ID:        state.ID,
		},
		ManagedModel: entutils.MapTimeMixinFromDB(state),
	}
}

// MarkSubscriptionChangedV2 joins the subscription mutation's transaction and
// advances only the current revision of an existing sync-state row.
func (a *adapter) MarkSubscriptionChangedV2(ctx context.Context, subscriptionID models.NamespacedID) error {
	if err := subscriptionID.Validate(); err != nil {
		return fmt.Errorf("validating changed subscription: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.ensureSubscriptionExists(ctx, subscriptionID); err != nil {
			return err
		}

		revision := ulid.Make().String()
		if err := tx.db.SubscriptionSyncStateV2.Create().
			SetNamespace(subscriptionID.Namespace).
			SetSubscriptionID(subscriptionID.ID).
			SetLastSuccessState(subscriptionsync.SyncStateV2StateNew).
			SetLastSyncedSubsRevision(revision).
			SetCurrentSubsRevision(revision).
			OnConflictColumns(subscriptionsyncstatev2.FieldNamespace, subscriptionsyncstatev2.FieldSubscriptionID).
			UpdateCurrentSubsRevision().
			UpdateUpdatedAt().
			Exec(ctx); err != nil {
			return fmt.Errorf("marking subscription changed for sync: %w", err)
		}

		return nil
	})
}

func (a *adapter) RecordSyncResultV2(ctx context.Context, input subscriptionsync.RecordSyncResultV2Input) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating subscription sync result v2: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.ensureSubscriptionExists(ctx, input.SubscriptionID); err != nil {
			return err
		}

		// Do not add UpdateCurrentSubsRevision here: it would overwrite a revision
		// written by a parallel subscription edit after sync read the state.
		err := tx.db.SubscriptionSyncStateV2.Create().
			SetNamespace(input.SubscriptionID.Namespace).
			SetSubscriptionID(input.SubscriptionID.ID).
			SetLastSuccessState(input.LastSuccessState).
			SetLastSuccessAt(input.LastSuccessAt.UTC()).
			SetNillableNextReconciliationAt(convert.SafeToUTC(input.NextReconciliationAt)).
			SetLastSyncedSubsRevision(input.LastSyncedSubsRevision).
			SetCurrentSubsRevision(input.LastSyncedSubsRevision).
			OnConflictColumns(subscriptionsyncstatev2.FieldNamespace, subscriptionsyncstatev2.FieldSubscriptionID).
			UpdateLastSuccessState().
			UpdateLastSuccessAt().
			// Include NULL so a terminal result clears the previous active deadline.
			UpdateNextReconciliationAt().
			UpdateUpdatedAt().
			UpdateLastSyncedSubsRevision().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("recording subscription sync result v2: %w", err)
		}

		return nil
	})
}

func (a *adapter) ensureSubscriptionExists(ctx context.Context, subscriptionID models.NamespacedID) error {
	// The foreign key checks the subscription ID; also enforce namespace ownership.
	// Include soft-deleted subscriptions because their cleanup is recorded here.
	exists, err := a.db.Subscription.Query().Where(
		dbsubscription.ID(subscriptionID.ID),
		dbsubscription.Namespace(subscriptionID.Namespace),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("checking subscription for sync state v2: %w", err)
	}

	if !exists {
		return subscription.NewSubscriptionNotFoundError(subscriptionID.ID)
	}

	return nil
}
