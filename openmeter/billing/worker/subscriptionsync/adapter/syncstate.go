package adapter

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionsyncstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)

func (a *adapter) GetSyncStates(ctx context.Context, input subscriptionsync.GetSyncStatesInput) ([]subscriptionsync.SyncStateManaged, error) {
	if len(input) == 0 {
		return []subscriptionsync.SyncStateManaged{}, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]subscriptionsync.SyncStateManaged, error) {
		requestedIDs := lo.SliceToMap(input, func(id models.NamespacedID) (models.NamespacedID, struct{}) {
			return id, struct{}{}
		})
		ids := lo.Map(input, func(id models.NamespacedID, _ int) string { return id.ID })
		states, err := tx.db.SubscriptionSyncState.Query().Where(subscriptionsyncstate.SubscriptionIDIn(ids...)).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting subscription sync states: %w", err)
		}

		// Subscription IDs are globally unique, but callers must also match the namespace.
		return lo.FilterMap(states, func(state *entdb.SubscriptionSyncState, _ int) (subscriptionsync.SyncStateManaged, bool) {
			mappedState := mapSyncStateFromDB(state)
			_, requested := requestedIDs[mappedState.SubscriptionID]

			return mappedState, requested
		}), nil
	})
}

func mapSyncStateFromDB(state *entdb.SubscriptionSyncState) subscriptionsync.SyncStateManaged {
	return subscriptionsync.SyncStateManaged{
		SyncState: subscriptionsync.SyncState{
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

// MarkSubscriptionChanged joins the subscription mutation's transaction and
// advances only the current revision of an existing sync-state row.
func (a *adapter) MarkSubscriptionChanged(ctx context.Context, subscriptionID models.NamespacedID) error {
	if err := subscriptionID.Validate(); err != nil {
		return fmt.Errorf("validating changed subscription: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.ensureSubscriptionExists(ctx, subscriptionID); err != nil {
			return err
		}

		revision := ulid.Make().String()
		if err := tx.db.SubscriptionSyncState.Create().
			SetNamespace(subscriptionID.Namespace).
			SetSubscriptionID(subscriptionID.ID).
			SetLastSuccessState(subscriptionsync.SyncStateStateNew).
			SetLastSyncedSubsRevision(revision).
			SetCurrentSubsRevision(revision).
			OnConflictColumns(subscriptionsyncstate.FieldNamespace, subscriptionsyncstate.FieldSubscriptionID).
			UpdateCurrentSubsRevision().
			UpdateUpdatedAt().
			Exec(ctx); err != nil {
			return fmt.Errorf("marking subscription changed for sync: %w", err)
		}

		return nil
	})
}

func (a *adapter) RecordSyncResult(ctx context.Context, input subscriptionsync.RecordSyncResultInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating subscription sync result: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.ensureSubscriptionExists(ctx, input.SubscriptionID); err != nil {
			return err
		}

		// Do not add UpdateCurrentSubsRevision here: it would overwrite a revision
		// written by a parallel subscription edit after sync read the state.
		err := tx.db.SubscriptionSyncState.Create().
			SetNamespace(input.SubscriptionID.Namespace).
			SetSubscriptionID(input.SubscriptionID.ID).
			SetLastSuccessState(input.LastSuccessState).
			SetLastSuccessAt(input.LastSuccessAt.UTC()).
			SetNillableNextReconciliationAt(convert.SafeToUTC(input.NextReconciliationAt)).
			SetLastSyncedSubsRevision(input.LastSyncedSubsRevision).
			SetCurrentSubsRevision(input.LastSyncedSubsRevision).
			OnConflictColumns(subscriptionsyncstate.FieldNamespace, subscriptionsyncstate.FieldSubscriptionID).
			UpdateLastSuccessState().
			UpdateLastSuccessAt().
			// Include NULL so a terminal result clears the previous active deadline.
			UpdateNextReconciliationAt().
			UpdateUpdatedAt().
			UpdateLastSyncedSubsRevision().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("recording subscription sync result: %w", err)
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
		return fmt.Errorf("checking subscription for sync state: %w", err)
	}

	if !exists {
		return subscription.NewSubscriptionNotFoundError(subscriptionID.ID)
	}

	return nil
}
