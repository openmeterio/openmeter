package reconciler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type subscriptionSyncSpy struct {
	subscriptionsync.Service
	options subscriptionsync.SynchronizeSubscriptionOptions
	synced  []models.NamespacedID
	// candidates and deleted are what the two selection queries return.
	candidates []models.NamespacedID
	deleted    []models.NamespacedID
	listInputs []subscriptionsync.ListSubscriptionsToReconcileInput
}

func (s *subscriptionSyncSpy) SyncByID(_ context.Context, id models.NamespacedID, _ time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	s.synced = append(s.synced, id)
	for _, opt := range opts {
		opt(&s.options)
	}

	return nil
}

func (s *subscriptionSyncSpy) ListSubscriptionsToReconcile(_ context.Context, input subscriptionsync.ListSubscriptionsToReconcileInput) (paginationv2.Result[models.NamespacedID], error) {
	s.listInputs = append(s.listInputs, input)

	return paginationv2.Result[models.NamespacedID]{Items: s.candidates}, nil
}

func (s *subscriptionSyncSpy) ListDeletedSubscriptionsToReconcile(context.Context, subscriptionsync.ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[models.NamespacedID], error) {
	return paginationv2.Result[models.NamespacedID]{Items: s.deleted}, nil
}

func TestAllSyncsEverySelectedSubscription(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// given: one subscription from the due query and one from the deletion cleanup query
	dueID := models.NamespacedID{Namespace: "namespace", ID: "due"}
	deletedID := models.NamespacedID{Namespace: "namespace", ID: "deleted"}
	spy := &subscriptionSyncSpy{
		candidates: []models.NamespacedID{dueID},
		deleted:    []models.NamespacedID{deletedID},
	}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: periodic reconciliation runs
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour},
	})

	// then: only due candidates were requested, and both selections are synced
	// without suppressing any billing behavior
	require.NoError(t, err)
	require.Len(t, spy.listInputs, 1)
	require.False(t, spy.listInputs[0].ForceSync)
	require.Equal(t, now, spy.listInputs[0].AsOf)
	require.Equal(t, time.Hour, spy.listInputs[0].Lookback)
	require.Equal(t, []models.NamespacedID{dueID, deletedID}, spy.synced)
	require.Equal(t, subscriptionsync.SynchronizeSubscriptionOptions{}, spy.options)
}

func TestAllForceSyncAsksForEverySubscription(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// given: a subscription the selection returns
	id := models.NamespacedID{Namespace: "namespace", ID: "not-due"}
	spy := &subscriptionSyncSpy{candidates: []models.NamespacedID{id}}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: reconciliation is forced, including subscriptions past the age cutoff
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour, ForceSync: true},
		ForceResyncOldSubscriptions:      true,
	})

	// then: selection is told to ignore sync state, and the sync bypasses the age cutoff
	require.NoError(t, err)
	require.Len(t, spy.listInputs, 1)
	require.True(t, spy.listInputs[0].ForceSync)
	require.Equal(t, []models.NamespacedID{id}, spy.synced)
	require.True(t, spy.options.ForceResyncOldSubscriptions)
}
