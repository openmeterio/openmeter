package subscriptionsync_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSyncStateV2SuccessTimestamp(t *testing.T) {
	for _, state := range []subscriptionsync.SyncStateV2State{
		subscriptionsync.SyncStateV2StateNew,
		subscriptionsync.SyncStateV2StateActive,
		subscriptionsync.SyncStateV2StateNoBillables,
		subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		subscriptionsync.SyncStateV2StateCustomerDeletedBefore,
		subscriptionsync.SyncStateV2StateEnded,
		subscriptionsync.SyncStateV2StateTooOld,
	} {
		t.Run(string(state), func(t *testing.T) {
			// New has no recorded result; every other state must have its result time.
			input := subscriptionsync.SyncStateV2{
				SubscriptionID:         models.NamespacedID{Namespace: "namespace", ID: "subscription"},
				LastSuccessState:       state,
				LastSyncedSubsRevision: "observed-revision",
				CurrentSubsRevision:    "current-revision",
			}
			if state == subscriptionsync.SyncStateV2StateActive {
				input.NextReconciliationAt = lo.ToPtr(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
			}

			if state == subscriptionsync.SyncStateV2StateNew {
				require.NoError(t, input.Validate())
				input.LastSuccessAt = lo.ToPtr(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
				require.ErrorContains(t, input.Validate(), "new sync state must not have a last success at")

				return
			}

			require.ErrorContains(t, input.Validate(), "last success at is required")
			input.LastSuccessAt = lo.ToPtr(time.Time{})
			require.ErrorContains(t, input.Validate(), "last success at is required")
			input.LastSuccessAt = lo.ToPtr(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
			require.NoError(t, input.Validate())
		})
	}
}

func TestActiveSyncStateRequiresNextReconciliation(t *testing.T) {
	at := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		next *time.Time
	}{
		{name: "missing deadline"},
		{name: "zero deadline", next: lo.ToPtr(time.Time{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Neither read models nor sync result writes may leave active subscriptions unscheduled.
			state := subscriptionsync.SyncStateV2{
				SubscriptionID:         models.NamespacedID{Namespace: "namespace", ID: "subscription"},
				LastSuccessState:       subscriptionsync.SyncStateV2StateActive,
				LastSuccessAt:          &at,
				NextReconciliationAt:   tc.next,
				LastSyncedSubsRevision: "observed-revision",
				CurrentSubsRevision:    "current-revision",
			}
			require.ErrorContains(t, state.Validate(), "next reconciliation at is required")
			require.ErrorContains(t, (subscriptionsync.RecordSyncResultV2Input{
				SubscriptionID:         state.SubscriptionID,
				LastSuccessState:       state.LastSuccessState,
				LastSuccessAt:          at,
				NextReconciliationAt:   tc.next,
				LastSyncedSubsRevision: state.LastSyncedSubsRevision,
			}).Validate(), "next reconciliation at is required")
		})
	}
}

func TestNonActiveSyncStateRejectsNextReconciliation(t *testing.T) {
	at := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	for _, resultState := range []subscriptionsync.SyncStateV2State{
		subscriptionsync.SyncStateV2StateNew,
		subscriptionsync.SyncStateV2StateNoBillables,
		subscriptionsync.SyncStateV2StateSubscriptionDeleted,
		subscriptionsync.SyncStateV2StateCustomerDeletedBefore,
		subscriptionsync.SyncStateV2StateEnded,
		subscriptionsync.SyncStateV2StateTooOld,
	} {
		t.Run(string(resultState), func(t *testing.T) {
			// Given a non-active state, no periodic reconciliation may be scheduled.
			state := subscriptionsync.SyncStateV2{
				SubscriptionID:         models.NamespacedID{Namespace: "namespace", ID: "subscription"},
				LastSuccessState:       resultState,
				LastSyncedSubsRevision: "observed-revision",
				CurrentSubsRevision:    "current-revision",
			}
			if resultState != subscriptionsync.SyncStateV2StateNew {
				state.LastSuccessAt = &at
			}

			input := subscriptionsync.RecordSyncResultV2Input{
				SubscriptionID:         state.SubscriptionID,
				LastSuccessState:       resultState,
				LastSuccessAt:          at,
				LastSyncedSubsRevision: state.LastSyncedSubsRevision,
			}
			require.NoError(t, state.Validate())
			if resultState == subscriptionsync.SyncStateV2StateNew {
				// Only subscription hooks may initialize new; recording success still rejects it.
				require.ErrorContains(t, input.Validate(), "new sync state must be initialized by a subscription change")
			} else {
				require.NoError(t, input.Validate())
			}

			for _, tc := range []struct {
				name string
				next time.Time
			}{
				{name: "nonzero deadline", next: at},
				{name: "zero deadline"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					// When a deadline is supplied, both validators must reject it, even if zero.
					stateWithDeadline := state
					stateWithDeadline.NextReconciliationAt = &tc.next
					inputWithDeadline := input
					inputWithDeadline.NextReconciliationAt = &tc.next
					require.ErrorContains(t, stateWithDeadline.Validate(), "next reconciliation at is only allowed for an active sync result")
					require.ErrorContains(t, inputWithDeadline.Validate(), "next reconciliation at is only allowed for an active sync result")
				})
			}
		})
	}
}
