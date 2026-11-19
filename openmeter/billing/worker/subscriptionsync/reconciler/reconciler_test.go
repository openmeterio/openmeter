package reconciler

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
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
	// candidatePages and deletedPages are what the two selection queries return,
	// one page per call.
	candidatePages [][]subscriptionsync.SubscriptionToReconcile
	deletedPages   [][]subscriptionsync.SubscriptionToReconcile
	listInputs     []subscriptionsync.ListSubscriptionsToReconcileInput
	// stuckCursor makes every page point back at itself.
	stuckCursor bool
	// onSync runs before a sync is recorded and can fail it.
	onSync func(models.NamespacedID) error
}

func (s *subscriptionSyncSpy) SyncByID(_ context.Context, id models.NamespacedID, _ time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	s.options = subscriptionsync.SynchronizeSubscriptionOptions{}
	for _, opt := range opts {
		opt(&s.options)
	}

	if s.onSync != nil {
		if err := s.onSync(id); err != nil {
			return err
		}
	}

	s.synced = append(s.synced, id)

	return nil
}

func (s *subscriptionSyncSpy) ListSubscriptionsToReconcile(_ context.Context, input subscriptionsync.ListSubscriptionsToReconcileInput) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
	s.listInputs = append(s.listInputs, input)

	return s.page(s.candidatePages, input.Cursor), nil
}

func (s *subscriptionSyncSpy) ListDeletedSubscriptionsToReconcile(_ context.Context, input subscriptionsync.ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
	return s.page(s.deletedPages, input.Cursor), nil
}

// page returns the page a cursor points at. The cursor carries the page index,
// so a page after the first is served only when the reconciler passes on the
// cursor it was given.
func (s *subscriptionSyncSpy) page(pages [][]subscriptionsync.SubscriptionToReconcile, cursor *paginationv2.Cursor) paginationv2.Result[subscriptionsync.SubscriptionToReconcile] {
	index := 0
	if cursor != nil {
		index, _ = strconv.Atoi(cursor.ID)
	}

	if index >= len(pages) {
		return paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{}
	}

	result := paginationv2.Result[subscriptionsync.SubscriptionToReconcile]{Items: pages[index]}

	next := index + 1
	if s.stuckCursor {
		next = index
	}

	if s.stuckCursor || next < len(pages) {
		nextCursor := paginationv2.NewCursor(time.Unix(int64(next), 0), strconv.Itoa(next))
		result.NextCursor = &nextCursor
	}

	return result
}

func candidate(id string) subscriptionsync.SubscriptionToReconcile {
	return subscriptionsync.SubscriptionToReconcile{
		NamespacedID: models.NamespacedID{Namespace: "namespace", ID: id},
		CustomerID:   "customer",
	}
}

func TestAllSyncsEverySelectedSubscription(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// given: two pages from the due query and one from the deletion cleanup query
	spy := &subscriptionSyncSpy{
		candidatePages: [][]subscriptionsync.SubscriptionToReconcile{{candidate("due-1")}, {candidate("due-2")}},
		deletedPages:   [][]subscriptionsync.SubscriptionToReconcile{{candidate("deleted")}},
	}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: periodic reconciliation runs
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour},
	})

	// then: every page is requested with the same selection time, and each
	// selected subscription is synced once
	require.NoError(t, err)
	require.Len(t, spy.listInputs, 2)
	for _, input := range spy.listInputs {
		require.False(t, input.ForceSync)
		require.Equal(t, now, input.AsOf)
		require.Equal(t, time.Hour, input.Lookback)
	}

	require.Nil(t, spy.listInputs[0].Cursor)
	require.NotNil(t, spy.listInputs[1].Cursor)
	require.Equal(t, []models.NamespacedID{
		candidate("due-1").NamespacedID,
		candidate("due-2").NamespacedID,
		candidate("deleted").NamespacedID,
	}, spy.synced)

	// and: a sync that an event finished after selection is left alone, without
	// suppressing any other billing behavior
	require.Equal(t, subscriptionsync.SynchronizeSubscriptionOptions{SkipIfSyncedSince: &now}, spy.options)
}

func TestAllForcedRunSyncsRegardlessOfOtherSyncs(t *testing.T) {
	// given: a forced run
	spy := &subscriptionSyncSpy{candidatePages: [][]subscriptionsync.SubscriptionToReconcile{{candidate("forced")}}}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: it syncs a subscription
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour, ForceSync: true},
	})

	// then: the sync is not skipped because another sync got there first
	require.NoError(t, err)
	require.True(t, spy.listInputs[0].ForceSync)
	require.Equal(t, []models.NamespacedID{candidate("forced").NamespacedID}, spy.synced)
	require.Nil(t, spy.options.SkipIfSyncedSince)
}

func TestAllKeepsGoingAfterAFailedSync(t *testing.T) {
	// given: three candidates, the second of which fails to sync
	syncErr := errors.New("billing is unavailable")
	spy := &subscriptionSyncSpy{
		candidatePages: [][]subscriptionsync.SubscriptionToReconcile{{candidate("first"), candidate("broken"), candidate("last")}},
		onSync: func(id models.NamespacedID) error {
			if id.ID == "broken" {
				return syncErr
			}

			return nil
		},
	}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: periodic reconciliation runs
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour},
	})

	// then: the others are synced, and the error names the subscription that failed
	require.ErrorIs(t, err, syncErr)
	require.ErrorContains(t, err, "namespace/broken")
	require.Equal(t, []models.NamespacedID{candidate("first").NamespacedID, candidate("last").NamespacedID}, spy.synced)
}

func TestAllStopsWhenTheRunIsCanceled(t *testing.T) {
	// given: a run that is canceled while its first subscription syncs
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	spy := &subscriptionSyncSpy{
		candidatePages: [][]subscriptionsync.SubscriptionToReconcile{{candidate("first"), candidate("second")}, {candidate("third")}},
		onSync: func(models.NamespacedID) error {
			cancel()

			return nil
		},
	}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: periodic reconciliation runs
	err := r.All(ctx, ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour},
	})

	// then: nothing else is synced and no further page is listed
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []models.NamespacedID{candidate("first").NamespacedID}, spy.synced)
	require.Len(t, spy.listInputs, 1)
}

func TestAllFailsWhenTheCursorDoesNotAdvance(t *testing.T) {
	// given: a candidate query that keeps returning the same cursor
	spy := &subscriptionSyncSpy{
		candidatePages: [][]subscriptionsync.SubscriptionToReconcile{{candidate("repeated")}},
		stuckCursor:    true,
	}
	r := Reconciler{subscriptionSync: spy, logger: slog.New(slog.DiscardHandler)}

	// when: periodic reconciliation runs
	err := r.All(t.Context(), ReconcilerAllInput{
		ReconcilerListSubscriptionsInput: ReconcilerListSubscriptionsInput{Lookback: time.Hour},
	})

	// then: the run ends with an error instead of listing the same page forever
	require.ErrorContains(t, err, "cursor did not advance")
	require.Len(t, spy.listInputs, 2)
}
