package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSyncStateV2Persistence(t *testing.T) {
	at := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

	t.Run("missing states", func(t *testing.T) {
		// given: a soft-deleted subscription with no recorded v2 reconciliation
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()

		// when: the subscription's state is requested
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)

		// then: no state is returned, including for an empty request
		require.Empty(t, states)
		states, err = f.adapter.GetSyncStatesV2(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, states)
	})

	t.Run("recording results preserves row identity and creation time", func(t *testing.T) {
		// given: a soft-deleted subscription with no recorded v2 reconciliation
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		ids := subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID}

		// when: successful cleanup is recorded with a non-UTC timestamp
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		states, err := f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Len(t, states, 1)
		createdState, err := f.db.SubscriptionSyncStateV2.Query().Only(ctx)
		require.NoError(t, err)

		// then: reads include the row identity and UTC timestamps
		expectedState := subscriptionsync.SyncStateV2Managed{
			SyncStateV2: subscriptionsync.SyncStateV2{
				SubscriptionID:         f.input.SubscriptionID,
				LastSuccessState:       f.input.LastSuccessState,
				LastSuccessAt:          lo.ToPtr(at),
				LastSyncedSubsRevision: f.input.LastSyncedSubsRevision,
				CurrentSubsRevision:    f.input.LastSyncedSubsRevision,
			},
			NamespacedID: models.NamespacedID{
				Namespace: f.input.SubscriptionID.Namespace,
				ID:        createdState.ID,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: at,
				UpdatedAt: at,
			},
		}
		require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)
		require.NoError(t, states[0].Validate())

		// when: cleanup is recorded again after its completion time
		clock.FreezeTime(at.Add(2 * time.Hour))
		defer clock.UnFreeze()
		input := f.input
		input.LastSuccessAt = at.Add(time.Hour)
		input.LastSuccessState = subscriptionsync.SyncStateV2StateCustomerDeletedBefore
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))

		// then: the same row holds the latest result and v1 state is untouched
		states, err = f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		expectedState.LastSuccessState = input.LastSuccessState
		expectedState.LastSuccessAt = lo.ToPtr(input.LastSuccessAt)
		expectedState.UpdatedAt = at.Add(2 * time.Hour)
		require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)
		count, err := f.db.SubscriptionSyncStateV2.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		legacyStates, err := f.adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, legacyStates)
	})

	t.Run("missing observed revision does not record cleanup", func(t *testing.T) {
		// Given cleanup did not supply the revision it observed under the billing lock.
		f := setupSyncStateV2PersistenceTest(t, at)
		input := f.input
		input.LastSyncedSubsRevision = ""

		// When recording success, the adapter rejects the missing revision instead of inventing one.
		require.ErrorContains(t, f.adapter.RecordSyncResultV2(t.Context(), input), "last synced subscription revision is required")

		// Then cleanup has not been acknowledged.
		states, err := f.adapter.GetSyncStatesV2(t.Context(), subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, states)
	})

	t.Run("cleanup preserves a newer current revision", func(t *testing.T) {
		// Given cleanup read one revision before another writer advanced it.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		observed, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, observed, 1)
		require.NoError(t, f.adapter.MarkSubscriptionChangedV2(ctx, f.input.SubscriptionID))
		changed, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, changed, 1)
		newerRevision := changed[0].CurrentSubsRevision
		require.NotEqual(t, observed[0].CurrentSubsRevision, newerRevision)

		// When cleanup acknowledges only the revision that it read.
		input := f.input
		input.LastSyncedSubsRevision = observed[0].CurrentSubsRevision
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))

		// Then the intervening change remains visible as two different revisions.
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, observed[0].CurrentSubsRevision, states[0].LastSyncedSubsRevision)
		require.Equal(t, newerRevision, states[0].CurrentSubsRevision)
	})

	t.Run("a row inserted after the initial read keeps its current revision", func(t *testing.T) {
		// Given cleanup found no row and generated its initial revision.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, states)

		// Another writer inserts a different revision before cleanup records success.
		newerRevision := ulid.Make().String()
		require.NoError(t, f.db.SubscriptionSyncStateV2.Create().
			SetNamespace(f.input.SubscriptionID.Namespace).
			SetSubscriptionID(f.input.SubscriptionID.ID).
			SetLastSuccessState(f.input.LastSuccessState).
			SetLastSuccessAt(at).
			SetLastSyncedSubsRevision(ulid.Make().String()).
			SetCurrentSubsRevision(newerRevision).Exec(ctx))
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))

		// Then ON CONFLICT acknowledges cleanup without replacing that writer's revision.
		states, err = f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, f.input.LastSyncedSubsRevision, states[0].LastSyncedSubsRevision)
		require.Equal(t, newerRevision, states[0].CurrentSubsRevision)
	})

	t.Run("namespace ownership", func(t *testing.T) {
		// given: a recorded state belonging to a different namespace
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		wrongNamespace := models.NamespacedID{Namespace: "other", ID: f.input.SubscriptionID.ID}

		// when: that subscription is read under the wrong namespace
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{wrongNamespace})
		require.NoError(t, err)

		// then: reads hide the state and writes report the subscription as missing
		require.Empty(t, states)
		invalidInput := f.input
		invalidInput.SubscriptionID = wrongNamespace
		require.ErrorAs(t, f.adapter.RecordSyncResultV2(ctx, invalidInput), new(*subscription.SubscriptionNotFoundError))
	})

	t.Run("transaction rollback", func(t *testing.T) {
		// given: a previously recorded successful reconciliation
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		ids := subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID}
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		expectedStates, err := f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Len(t, expectedStates, 1)

		// when: a later reconciliation writes its result but its transaction fails
		clock.FreezeTime(at.Add(3 * time.Hour))
		defer clock.UnFreeze()
		rollbackErr := errors.New("reconciliation failed")
		err = transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
			pending := f.input
			pending.LastSuccessAt = at.Add(2 * time.Hour)
			pending.LastSyncedSubsRevision = ulid.Make().String()
			if err := f.adapter.RecordSyncResultV2(ctx, pending); err != nil {
				return err
			}

			return rollbackErr
		})
		require.ErrorIs(t, err, rollbackErr)

		// then: the previous result and its row timestamps remain intact
		states, err := f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Equal(t, expectedStates, states)
	})

	t.Run("foreign key rejects missing subscription", func(t *testing.T) {
		// given: no subscription exists for the referenced ID
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()

		// when: a database write bypasses the adapter's subscription lookup
		err := f.db.SubscriptionSyncStateV2.Create().SetNamespace(f.input.SubscriptionID.Namespace).
			SetSubscriptionID(ulid.Make().String()).SetLastSuccessState(f.input.LastSuccessState).
			SetLastSyncedSubsRevision(f.input.LastSyncedSubsRevision).
			SetCurrentSubsRevision(f.input.LastSyncedSubsRevision).
			SetLastSuccessAt(at).Exec(ctx)

		// then: the foreign key rejects the state
		require.True(t, entdb.IsConstraintError(err), "%v", err)
	})

	t.Run("hard deletion cascades to sync state", func(t *testing.T) {
		// given: a soft-deleted subscription retains its recorded sync state
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		ids := subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID}
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		states, err := f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Len(t, states, 1)

		// when: the parent subscription is hard-deleted
		require.NoError(t, f.db.Subscription.DeleteOneID(f.input.SubscriptionID.ID).Exec(ctx))

		// then: its sync state is removed too
		states, err = f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Empty(t, states)
	})
}

func TestMarkSubscriptionChangedV2(t *testing.T) {
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	t.Run("missing row initializes new state without claiming a result", func(t *testing.T) {
		// Given a subscription that predates v2 state tracking.
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()

		// When its change is marked, the adapter inserts a new state.
		require.NoError(t, f.adapter.MarkSubscriptionChangedV2(ctx, f.input.SubscriptionID))
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		before := states[0]
		require.Equal(t, subscriptionsync.SyncStateV2StateNew, before.LastSuccessState)
		require.Nil(t, before.LastSuccessAt)
		require.NotEmpty(t, before.CurrentSubsRevision)
		require.Equal(t, before.CurrentSubsRevision, before.LastSyncedSubsRevision)
		require.NoError(t, before.Validate())

		// Then a second change advances current without acknowledging either revision.
		require.NoError(t, f.adapter.MarkSubscriptionChangedV2(ctx, f.input.SubscriptionID))
		states, err = f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, before.ID, states[0].ID)
		require.Equal(t, subscriptionsync.SyncStateV2StateNew, states[0].LastSuccessState)
		require.Nil(t, states[0].LastSuccessAt)
		require.Equal(t, before.LastSyncedSubsRevision, states[0].LastSyncedSubsRevision)
		require.NotEqual(t, before.CurrentSubsRevision, states[0].CurrentSubsRevision)
	})

	t.Run("existing result is preserved", func(t *testing.T) {
		// Given a recorded result whose revision was acknowledged.
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		before := states[0]
		clock.FreezeTime(at.Add(time.Hour))
		defer clock.UnFreeze()

		// When the subscription changes, only current and the row update time advance.
		require.NoError(t, f.adapter.MarkSubscriptionChangedV2(ctx, f.input.SubscriptionID))
		states, err = f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, before.ID, states[0].ID)
		require.Equal(t, before.CreatedAt, states[0].CreatedAt)
		require.Equal(t, at.Add(time.Hour), states[0].UpdatedAt)
		require.Equal(t, before.LastSuccessState, states[0].LastSuccessState)
		require.Equal(t, before.LastSuccessAt, states[0].LastSuccessAt)
		require.Equal(t, before.LastSyncedSubsRevision, states[0].LastSyncedSubsRevision)
		require.NotEqual(t, before.CurrentSubsRevision, states[0].CurrentSubsRevision)
	})

	t.Run("namespace ownership and missing subscriptions return not found", func(t *testing.T) {
		// Given a subscription that belongs to one namespace.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()

		// When a caller uses another namespace or a missing ID, neither write is allowed.
		for _, id := range []models.NamespacedID{
			{Namespace: "other", ID: f.input.SubscriptionID.ID},
			{Namespace: f.input.SubscriptionID.Namespace, ID: ulid.Make().String()},
		} {
			require.ErrorAs(t, f.adapter.MarkSubscriptionChangedV2(ctx, id), new(*subscription.SubscriptionNotFoundError))
		}

		// Then no sync-state row has been created.
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, states)
	})

	t.Run("revision changes join the caller transaction", func(t *testing.T) {
		// Given an existing result and a mutation that will roll back.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, f.input))
		before, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		rollbackErr := errors.New("subscription mutation failed")

		// When the outer transaction fails after marking the change.
		err = transaction.RunWithNoValue(ctx, f.adapter, func(ctx context.Context) error {
			if err := f.adapter.MarkSubscriptionChangedV2(ctx, f.input.SubscriptionID); err != nil {
				return err
			}

			return rollbackErr
		})
		require.ErrorIs(t, err, rollbackErr)

		// Then the previous result and revision marker remain intact.
		after, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID})
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	t.Run("result recording cannot initialize new state", func(t *testing.T) {
		// Given the result writer, which must only acknowledge completed outcomes.
		f := setupSyncStateV2PersistenceTest(t, at)
		input := f.input
		input.LastSuccessState = subscriptionsync.SyncStateV2StateNew

		// When a caller tries to record new as a completed result, validation rejects it.
		require.ErrorContains(t, f.adapter.RecordSyncResultV2(t.Context(), input), "initialized by a subscription change")
		states, err := f.adapter.GetSyncStatesV2(t.Context(), subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, states)
	})
}

type syncStateV2PersistenceFixture struct {
	adapter subscriptionsync.Adapter
	db      *entdb.Client
	input   subscriptionsync.RecordSyncResultV2Input
}

func setupSyncStateV2PersistenceTest(t *testing.T, at time.Time) syncStateV2PersistenceFixture {
	t.Helper()
	ctx := t.Context()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	db := testDB.EntDriver.Client()
	t.Cleanup(func() {
		testDB.Close(t)
	})
	adapter, err := syncadapter.New(syncadapter.Config{Client: db})
	require.NoError(t, err)
	namespace := ulid.Make().String()
	customer, err := db.Customer.Create().SetNamespace(namespace).SetName("Sync state customer").Save(ctx)
	require.NoError(t, err)
	repository := subscriptionrepo.NewSubscriptionRepo(db)
	sub, err := repository.Create(ctx, subscription.CreateSubscriptionEntityInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		CadencedModel:   models.CadencedModel{ActiveFrom: at.Add(24 * time.Hour)},
		Name:            "Scheduled subscription",
		CustomerId:      customer.ID,
		InvoiceCurrency: "USD",
		BillingCadence:  datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0),
		BillingAnchor:   at,
		SettlementMode:  productcatalog.CreditThenInvoiceSettlementMode,
	})
	require.NoError(t, err)
	require.NoError(t, repository.Delete(ctx, sub.NamespacedID))

	return syncStateV2PersistenceFixture{
		adapter: adapter,
		db:      db,
		input: subscriptionsync.RecordSyncResultV2Input{
			SubscriptionID:         sub.NamespacedID,
			LastSuccessState:       subscriptionsync.SyncStateV2StateSubscriptionDeleted,
			LastSuccessAt:          at.In(time.FixedZone("test", 2*60*60)),
			LastSyncedSubsRevision: ulid.Make().String(),
		},
	}
}

func TestActiveSyncStateV2Persistence(t *testing.T) {
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	t.Run("active deadline survives subscription changes", func(t *testing.T) {
		// Given an active result with a deadline provided in another time zone.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		input := f.input
		input.LastSuccessState = subscriptionsync.SyncStateV2StateActive
		next := at.Add(24 * time.Hour).In(time.FixedZone("test", 2*60*60))
		input.NextReconciliationAt = &next
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, lo.ToPtr(next.UTC()), states[0].NextReconciliationAt)
		require.NoError(t, states[0].Validate())
		before := states[0]

		// When a subscription edit advances current, the previous deadline remains intact.
		require.NoError(t, f.adapter.MarkSubscriptionChangedV2(ctx, input.SubscriptionID))
		states, err = f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, before.NextReconciliationAt, states[0].NextReconciliationAt)
		require.Equal(t, before.LastSyncedSubsRevision, states[0].LastSyncedSubsRevision)
		require.NotEqual(t, before.CurrentSubsRevision, states[0].CurrentSubsRevision)

		// Then the next successful reconciliation replaces the deadline and acknowledges that edit.
		input.LastSyncedSubsRevision = states[0].CurrentSubsRevision
		input.NextReconciliationAt = lo.ToPtr(at.Add(48 * time.Hour))
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))
		states, err = f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		require.Equal(t, input.NextReconciliationAt, states[0].NextReconciliationAt)
		require.Equal(t, states[0].CurrentSubsRevision, states[0].LastSyncedSubsRevision)
	})

	for _, unscheduledState := range []subscriptionsync.SyncStateV2State{
		subscriptionsync.SyncStateV2StateNoBillables,
		subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		subscriptionsync.SyncStateV2StateCustomerDeletedBefore,
		subscriptionsync.SyncStateV2StateEnded,
		subscriptionsync.SyncStateV2StateTooOld,
	} {
		t.Run(string(unscheduledState)+" clears the active deadline", func(t *testing.T) {
			// Given an active schedule, when a result with no further periodic reconciliation replaces it.
			f := setupSyncStateV2PersistenceTest(t, at)
			ctx := t.Context()
			input := f.input
			input.LastSuccessState = subscriptionsync.SyncStateV2StateActive
			input.NextReconciliationAt = lo.ToPtr(at.Add(24 * time.Hour))
			require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))
			input.LastSuccessState = unscheduledState
			input.NextReconciliationAt = nil
			require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))

			// Then the upsert must clear the nullable deadline on conflict.
			states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
			require.NoError(t, err)
			require.Len(t, states, 1)
			require.Equal(t, unscheduledState, states[0].LastSuccessState)
			require.Nil(t, states[0].NextReconciliationAt)
		})
	}

	t.Run("PostgreSQL rejects an unscheduled active row", func(t *testing.T) {
		// Given a write that bypasses the domain validator.
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		err := f.db.SubscriptionSyncStateV2.Create().
			SetNamespace(f.input.SubscriptionID.Namespace).
			SetSubscriptionID(f.input.SubscriptionID.ID).
			SetLastSuccessState(subscriptionsync.SyncStateV2StateActive).
			SetLastSuccessAt(at).
			SetLastSyncedSubsRevision(f.input.LastSyncedSubsRevision).
			SetCurrentSubsRevision(f.input.LastSyncedSubsRevision).Exec(ctx)

		// Then the database also prevents silently losing active subscriptions from polling.
		require.ErrorContains(t, err, "subscription_sync_v2_active_reconciliation_at")
		input := f.input
		input.LastSuccessState = subscriptionsync.SyncStateV2StateActive
		input.NextReconciliationAt = lo.ToPtr(at.Add(time.Hour))
		require.NoError(t, f.adapter.RecordSyncResultV2(ctx, input))
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{input.SubscriptionID})
		require.NoError(t, err)
		require.Len(t, states, 1)
		err = f.db.SubscriptionSyncStateV2.UpdateOneID(states[0].ID).ClearNextReconciliationAt().Exec(ctx)
		require.ErrorContains(t, err, "subscription_sync_v2_active_reconciliation_at")
	})
}
