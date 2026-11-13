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
	ctx := t.Context()
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	db := testDB.EntDriver.Client()
	t.Cleanup(func() {
		testDB.Close(t)
	})
	adapter, err := syncadapter.New(syncadapter.Config{Client: db})
	require.NoError(t, err)

	// given: a soft-deleted subscription with no recorded v2 reconciliation
	at := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(at)
	defer clock.UnFreeze()

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
	ids := subscriptionsync.GetSyncStatesV2Input{sub.NamespacedID}
	states, err := adapter.GetSyncStatesV2(ctx, ids)
	require.NoError(t, err)
	require.Empty(t, states)
	states, err = adapter.GetSyncStatesV2(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, states)

	// when: successful cleanup is recorded and subsequently recorded again
	input := subscriptionsync.UpsertSyncStateV2Input{
		SubscriptionID:   sub.NamespacedID,
		LastSuccessState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		LastSuccessAt:    at.In(time.FixedZone("test", 2*60*60)),
	}
	require.NoError(t, adapter.UpsertSyncStateV2(ctx, input))
	states, err = adapter.GetSyncStatesV2(ctx, ids)
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Equal(t, at, states[0].LastSuccessAt)
	createdState, err := db.SubscriptionSyncStateV2.Query().Only(ctx)
	require.NoError(t, err)
	expectedState := subscriptionsync.SyncStateV2Managed{
		SyncStateV2: input,
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
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

	clock.FreezeTime(at.Add(2 * time.Hour))
	defer clock.UnFreeze()

	input.LastSuccessAt = at.Add(time.Hour)
	require.NoError(t, adapter.UpsertSyncStateV2(ctx, input))

	// then: the same row contains the latest successful result and v1 state is untouched
	states, err = adapter.GetSyncStatesV2(ctx, ids)
	require.NoError(t, err)
	expectedState.SyncStateV2 = input
	expectedState.UpdatedAt = at.Add(2 * time.Hour)
	require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)
	count, err := db.SubscriptionSyncStateV2.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	legacyStates, err := adapter.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{sub.NamespacedID})
	require.NoError(t, err)
	require.Empty(t, legacyStates)

	// given: the same subscription ID is requested under another namespace
	wrongNamespace := models.NamespacedID{Namespace: "other", ID: sub.ID}
	states, err = adapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{wrongNamespace})
	require.NoError(t, err)
	require.Empty(t, states)
	invalidInput := input
	invalidInput.SubscriptionID = wrongNamespace
	require.ErrorAs(t, adapter.UpsertSyncStateV2(ctx, invalidInput), new(*subscription.SubscriptionNotFoundError))

	// when: a reconciliation writes its result but the surrounding transaction fails
	clock.FreezeTime(at.Add(3 * time.Hour))
	defer clock.UnFreeze()

	rollbackErr := errors.New("reconciliation failed")
	err = transaction.RunWithNoValue(ctx, adapter, func(ctx context.Context) error {
		pending := input
		pending.LastSuccessAt = at.Add(2 * time.Hour)
		if err := adapter.UpsertSyncStateV2(ctx, pending); err != nil {
			return err
		}

		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	// then: the previous successful result remains intact
	states, err = adapter.GetSyncStatesV2(ctx, ids)
	require.NoError(t, err)
	require.Equal(t, []subscriptionsync.SyncStateV2Managed{expectedState}, states)

	// A database write cannot reference a nonexistent subscription.
	err = db.SubscriptionSyncStateV2.Create().SetNamespace(namespace).
		SetSubscriptionID(ulid.Make().String()).SetLastSuccessState(input.LastSuccessState).
		SetLastSuccessAt(at).Exec(ctx)
	require.True(t, entdb.IsConstraintError(err), "%v", err)

	// Hard-deleting the parent removes its state; soft deletion above retained it.
	require.NoError(t, db.Subscription.DeleteOneID(sub.ID).Exec(ctx))
	states, err = adapter.GetSyncStatesV2(ctx, ids)
	require.NoError(t, err)
	require.Empty(t, states)
}
