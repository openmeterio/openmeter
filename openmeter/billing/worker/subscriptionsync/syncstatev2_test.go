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
