package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestReconcileSubscriptionUsesDefaultSyncOptions(t *testing.T) {
	// given:
	// - periodic reconciliation delegates to subscription sync
	// when:
	// - a subscription is reconciled
	// then:
	// - it does not suppress any supported billing behavior
	spy := &subscriptionSyncSpy{}
	reconciler := Reconciler{subscriptionSync: spy}

	err := reconciler.ReconcileSubscription(t.Context(), models.NamespacedID{
		Namespace: "namespace",
		ID:        "subscription-id",
	})

	require.NoError(t, err)
	require.Equal(t, subscriptionsync.SynchronizeSubscriptionOptions{}, spy.options)
}

type subscriptionSyncSpy struct {
	subscriptionsync.Service
	options      subscriptionsync.SynchronizeSubscriptionOptions
	legacyStates []subscriptionsync.SyncState
	statesV2     []subscriptionsync.SyncStateV2Managed
	legacyIDs    subscriptionsync.GetSyncStatesInput
	v2IDs        subscriptionsync.GetSyncStatesV2Input
}

func (s *subscriptionSyncSpy) SyncByID(_ context.Context, _ models.NamespacedID, _ time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	for _, opt := range opts {
		opt(&s.options)
	}

	return nil
}

func (s *subscriptionSyncSpy) GetSyncStates(_ context.Context, ids subscriptionsync.GetSyncStatesInput) ([]subscriptionsync.SyncState, error) {
	s.legacyIDs = ids
	return s.legacyStates, nil
}

func (s *subscriptionSyncSpy) GetSyncStatesV2(_ context.Context, ids subscriptionsync.GetSyncStatesV2Input) ([]subscriptionsync.SyncStateV2Managed, error) {
	s.v2IDs = ids
	return s.statesV2, nil
}

func TestSubscriptionWithSyncStateShouldReconcile(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	completedDeletion := &subscriptionsync.SyncStateV2Managed{
		SyncStateV2: subscriptionsync.SyncStateV2{
			LastSuccessState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		},
	}
	tests := []struct {
		name      string
		deletedAt *time.Time
		state     *subscriptionsync.SyncState
		stateV2   *subscriptionsync.SyncStateV2Managed
		want      bool
	}{
		{
			name: "no previous sync",
			// A subscription without a legacy checkpoint still needs its initial sync.
			want: true,
		},
		{
			name:  "no billables",
			state: &subscriptionsync.SyncState{NextSyncAfter: &past},
			// A stale deadline does not schedule a subscription with no billables.
			want: false,
		},
		{
			name:  "billables without deadline",
			state: &subscriptionsync.SyncState{HasBillables: true},
			// A nil deadline continues to mean no further periodic sync is needed.
			want: false,
		},
		{
			name: "past billing deadline",
			state: &subscriptionsync.SyncState{
				HasBillables:  true,
				NextSyncAfter: &past,
			},
			want: true,
		},
		{
			name: "billing deadline is now",
			state: &subscriptionsync.SyncState{
				HasBillables:  true,
				NextSyncAfter: &now,
			},
			want: true,
		},
		{
			name: "future billing deadline",
			state: &subscriptionsync.SyncState{
				HasBillables:  true,
				NextSyncAfter: &future,
			},
			want: false,
		},
		{
			name: "deleted without v2 cleanup",
			// Legacy no-billables state cannot establish that deletion was reconciled.
			deletedAt: &past,
			state:     &subscriptionsync.SyncState{},
			want:      true,
		},
		{
			name:      "deletion takes effect now",
			deletedAt: &now,
			want:      true,
		},
		{
			name:      "deleted with completed cleanup",
			deletedAt: &past,
			state: &subscriptionsync.SyncState{
				HasBillables:  true,
				NextSyncAfter: &past,
			},
			stateV2: completedDeletion,
			// Completed cleanup takes precedence over the old billing deadline.
			want: false,
		},
		{
			name:      "deleted with a different v2 result",
			deletedAt: &past,
			stateV2: &subscriptionsync.SyncStateV2Managed{
				SyncStateV2: subscriptionsync.SyncStateV2{
					LastSuccessState: "active",
				},
			},
			// A future v2 result for active billing does not establish deletion cleanup.
			want: true,
		},
		{
			name:      "future deletion without any sync state",
			deletedAt: &future,
			want:      true,
		},
		{
			name:      "future deletion ignores a future billing deadline",
			deletedAt: &future,
			state:     &subscriptionsync.SyncState{HasBillables: true, NextSyncAfter: &future},
			want:      true,
		},
		{
			name:      "future deletion with another v2 result",
			deletedAt: &future,
			stateV2: &subscriptionsync.SyncStateV2Managed{
				SyncStateV2: subscriptionsync.SyncStateV2{LastSuccessState: "active"},
			},
			want: true,
		},
		{
			name: "future deletion requires cleanup despite legacy no billables",
			// Deletion is immediate even when its timestamp is ahead of the worker clock.
			deletedAt: &future,
			state:     &subscriptionsync.SyncState{},
			want:      true,
		},
		{
			name:      "future deletion with completed cleanup ignores due billing",
			deletedAt: &future,
			state: &subscriptionsync.SyncState{
				HasBillables:  true,
				NextSyncAfter: &past,
			},
			stateV2: completedDeletion,
			// Cleanup already removed the future gathering lines; do not schedule them again.
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub := SubscriptionWithSyncState{
				Subscription: subscription.Subscription{
					ManagedModel: models.ManagedModel{DeletedAt: tc.deletedAt},
				},
				SyncState:   tc.state,
				SyncStateV2: tc.stateV2,
			}
			require.Equal(t, tc.want, sub.ShouldReconcile(now))
		})
	}
}

func TestMapToSubscriptionWithSyncStateIncludesV2(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// given: cleanup finished after the deleted subscription was selected
	activeID := models.NamespacedID{Namespace: "namespace", ID: "active"}
	deletedID := models.NamespacedID{Namespace: "namespace", ID: "deleted"}
	missingID := models.NamespacedID{Namespace: "namespace", ID: "missing"}
	legacy := subscriptionsync.SyncState{
		SubscriptionID: activeID,
		HasBillables:   true,
		NextSyncAfter:  &now,
	}
	completed := subscriptionsync.SyncStateV2Managed{
		SyncStateV2: subscriptionsync.SyncStateV2{
			SubscriptionID:   deletedID,
			LastSuccessState: subscriptionsync.SyncStateV2StateSubscriptionDeleted,
			LastSuccessAt:    now,
		},
		NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "state-row-id"},
		ManagedModel: models.ManagedModel{CreatedAt: now, UpdatedAt: now},
	}
	activeStateV2 := completed
	activeStateV2.SubscriptionID = activeID
	activeStateV2.ID = "active-state-row"
	foreignStateV2 := completed
	foreignStateV2.SubscriptionID = models.NamespacedID{Namespace: "other", ID: missingID.ID}
	foreignStateV2.NamespacedID = models.NamespacedID{Namespace: "other", ID: "foreign-state-row"}
	spy := &subscriptionSyncSpy{
		legacyStates: []subscriptionsync.SyncState{legacy},
		statesV2:     []subscriptionsync.SyncStateV2Managed{completed, activeStateV2, foreignStateV2},
	}
	r := Reconciler{subscriptionSync: spy}

	// when: both state versions are attached by subscription identity
	mapped, err := r.mapToSubscriptionWithSyncState(t.Context(), []subscription.Subscription{
		{NamespacedID: activeID},
		{
			NamespacedID: deletedID,
			ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(now.Add(2 * time.Second))},
		},
		{NamespacedID: missingID},
	})

	// then: non-deleted subscriptions retain legacy scheduling and v2 metadata
	require.NoError(t, err)
	require.Len(t, mapped, 3)
	require.Equal(t, &legacy, mapped[0].SyncState)
	require.Equal(t, &activeStateV2, mapped[0].SyncStateV2)
	require.True(t, mapped[0].ShouldReconcile(now))
	// And a deleted subscription is not suppressed by legacy state or requeued after cleanup.
	require.Nil(t, mapped[1].SyncState)
	require.Equal(t, &completed, mapped[1].SyncStateV2)
	require.False(t, mapped[1].ShouldReconcile(now))
	require.Nil(t, mapped[2].SyncStateV2, "sync state from another namespace must not match")
	require.Equal(t, subscriptionsync.GetSyncStatesInput{activeID, missingID}, spy.legacyIDs)
	require.Equal(t, subscriptionsync.GetSyncStatesV2Input{activeID, deletedID, missingID}, spy.v2IDs)
}
