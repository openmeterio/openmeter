package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
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

	t.Run("upsert preserves row identity and creation time", func(t *testing.T) {
		// given: a soft-deleted subscription with no recorded v2 reconciliation
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		ids := subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID}

		// when: successful cleanup is recorded with a non-UTC timestamp
		require.NoError(t, f.adapter.UpsertSyncStateV2(ctx, f.input))
		states, err := f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		require.Len(t, states, 1)
		createdState, err := f.db.SubscriptionSyncStateV2.Query().Only(ctx)
		require.NoError(t, err)

		// then: reads include the row identity and UTC timestamps
		expectedState := subscriptionsync.SyncStateV2Managed{
			SyncStateV2: f.input,
			NamespacedID: models.NamespacedID{
				Namespace: f.input.SubscriptionID.Namespace,
				ID:        createdState.ID,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: at,
				UpdatedAt: at,
			},
		}
		expectedState.LastSuccessAt = at
		require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)
		require.NoError(t, states[0].Validate())

		// when: cleanup is recorded again after its completion time
		clock.FreezeTime(at.Add(2 * time.Hour))
		defer clock.UnFreeze()
		input := f.input
		input.LastSuccessAt = at.Add(time.Hour)
		require.NoError(t, f.adapter.UpsertSyncStateV2(ctx, input))

		// then: the same row holds the latest result and v1 state is untouched
		states, err = f.adapter.GetSyncStatesV2(ctx, ids)
		require.NoError(t, err)
		expectedState.SyncStateV2 = input
		expectedState.UpdatedAt = at.Add(2 * time.Hour)
		require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)
		count, err := f.db.SubscriptionSyncStateV2.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		legacyStates, err := f.adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{input.SubscriptionID})
		require.NoError(t, err)
		require.Empty(t, legacyStates)
	})

	t.Run("namespace ownership", func(t *testing.T) {
		// given: a recorded state belonging to a different namespace
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		require.NoError(t, f.adapter.UpsertSyncStateV2(ctx, f.input))
		wrongNamespace := models.NamespacedID{Namespace: "other", ID: f.input.SubscriptionID.ID}

		// when: that subscription is read under the wrong namespace
		states, err := f.adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{wrongNamespace})
		require.NoError(t, err)

		// then: reads hide the state and writes report the subscription as missing
		require.Empty(t, states)
		invalidInput := f.input
		invalidInput.SubscriptionID = wrongNamespace
		require.ErrorAs(t, f.adapter.UpsertSyncStateV2(ctx, invalidInput), new(*subscription.SubscriptionNotFoundError))
	})

	t.Run("transaction rollback", func(t *testing.T) {
		// given: a previously recorded successful reconciliation
		clock.FreezeTime(at)
		defer clock.UnFreeze()
		f := setupSyncStateV2PersistenceTest(t, at)
		ctx := t.Context()
		ids := subscriptionsync.GetSyncStatesV2Input{f.input.SubscriptionID}
		require.NoError(t, f.adapter.UpsertSyncStateV2(ctx, f.input))
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
			if err := f.adapter.UpsertSyncStateV2(ctx, pending); err != nil {
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
		require.NoError(t, f.adapter.UpsertSyncStateV2(ctx, f.input))
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

type syncStateV2PersistenceFixture struct {
	adapter subscriptionsync.Adapter
	db      *entdb.Client
	input   subscriptionsync.UpsertSyncStateV2Input
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
		input: subscriptionsync.UpsertSyncStateV2Input{
			SubscriptionID:   sub.NamespacedID,
			LastSuccessState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
			LastSuccessAt:    at.In(time.FixedZone("test", 2*60*60)),
		},
	}
}
