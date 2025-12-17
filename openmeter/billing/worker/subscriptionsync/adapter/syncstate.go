package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionbillingsyncstate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)

func (a *adapter) InvalidateSyncState(ctx context.Context, input subscriptionsync.InvalidateSyncStateInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.SubscriptionBillingSyncState.Delete().
			Where(subscriptionbillingsyncstate.SubscriptionID(input.ID)).
			Where(subscriptionbillingsyncstate.Namespace(input.Namespace)).
			Exec(ctx)

		return err
	})
}

func (a *adapter) GetSyncStates(ctx context.Context, input subscriptionsync.GetSyncStatesInput) ([]subscriptionsync.SyncState, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]subscriptionsync.SyncState, error) {
		res, err := tx.db.SubscriptionBillingSyncState.Query().
			Where(subscriptionbillingsyncstate.SubscriptionIDIn(lo.Map(input, func(id models.NamespacedID, _ int) string {
				return id.ID
			})...)).All(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(res, func(state *entdb.SubscriptionBillingSyncState, _ int) subscriptionsync.SyncState {
			return mapSyncStateFromDB(state)
		}), nil
	})
}

func mapSyncStateFromDB(state *entdb.SubscriptionBillingSyncState) subscriptionsync.SyncState {
	nextSyncAfter := state.NextSyncAfter
	if nextSyncAfter != nil {
		nextSyncAfter = lo.ToPtr(nextSyncAfter.UTC())
	}

	return subscriptionsync.SyncState{
		SubscriptionID: models.NamespacedID{ID: state.SubscriptionID, Namespace: state.Namespace},
		HasBillables:   state.HasBillables,
		SyncedAt:       state.SyncedAt.UTC(),
		NextSyncAfter:  nextSyncAfter,
	}
}

func (a *adapter) UpsertSyncState(ctx context.Context, input subscriptionsync.UpsertSyncStateInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		nextSyncAfter := input.NextSyncAfter
		if nextSyncAfter != nil {
			nextSyncAfter = lo.ToPtr(nextSyncAfter.UTC())
		}

		return tx.db.SubscriptionBillingSyncState.Create().
			SetHasBillables(input.HasBillables).
			SetSyncedAt(input.SyncedAt.UTC()).
			SetNillableNextSyncAfter(nextSyncAfter).
			SetSubscriptionID(input.SubscriptionID.ID).
			SetNamespace(input.SubscriptionID.Namespace).
			OnConflictColumns(
				subscriptionbillingsyncstate.FieldSubscriptionID,
				subscriptionbillingsyncstate.FieldNamespace,
			).
			UpdateHasBillables().
			UpdateSyncedAt().
			UpdateNextSyncAfter().
			Exec(ctx)
	})
}
