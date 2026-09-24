package meta

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestPatchUpdateSubscriptionReference(t *testing.T) {
	phaseID := "phase-2"
	subscriptionItemID := "item-2"

	patch, err := NewPatchUpdateSubscriptionReference(NewPatchUpdateSubscriptionReferenceInput{
		PhaseID:            &phaseID,
		SubscriptionItemID: &subscriptionItemID,
	})

	require.NoError(t, err)
	phaseID = "mutated-by-caller"
	subscriptionItemID = "mutated-by-caller"
	updated, err := patch.Apply(SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
	})
	require.NoError(t, err)
	require.Equal(t, SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-2",
		ItemID:         "item-2",
	}, updated)
	require.Equal(t, PatchTypeUpdateSubscriptionReference, patch.Op())
	require.Equal(t, TriggerUpdateSubscriptionReference, patch.Trigger())
	target, err := patch.GetTargetLayer(nil)
	require.NoError(t, err)
	require.Equal(t, ChangeTargetBase, target)
}

func TestPatchUpdateSubscriptionReferenceValidation(t *testing.T) {
	tests := []struct {
		name               string
		phaseID            *string
		subscriptionItemID *string
		errorString        string
	}{
		{
			name:        "empty update",
			errorString: "at least one subscription reference update is required",
		},
		{
			name:        "empty phase ID",
			phaseID:     lo.ToPtr(""),
			errorString: "phase ID is required",
		},
		{
			name:               "empty subscription item ID",
			subscriptionItemID: lo.ToPtr(""),
			errorString:        "subscription item ID is required",
		},
		{
			name:               "both values are invalid",
			phaseID:            lo.ToPtr(""),
			subscriptionItemID: lo.ToPtr(""),
			errorString:        "phase ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPatchUpdateSubscriptionReference(NewPatchUpdateSubscriptionReferenceInput{
				PhaseID:            tt.phaseID,
				SubscriptionItemID: tt.subscriptionItemID,
			})

			require.ErrorContains(t, err, tt.errorString)
		})
	}
}

func TestSubscriptionReferenceAsPatchUpdateSubscriptionReference(t *testing.T) {
	existing := SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
	}
	target := SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-2",
		ItemID:         "item-2",
	}

	patch, err := target.AsPatchUpdateSubscriptionReference(existing)
	require.NoError(t, err)

	updated, err := patch.Apply(existing)
	require.NoError(t, err)
	require.Equal(t, target, updated)

	t.Run("subscription ID cannot change", func(t *testing.T) {
		target := target
		target.SubscriptionID = "subscription-2"

		_, err := target.AsPatchUpdateSubscriptionReference(existing)
		require.ErrorContains(t, err, "subscription ID cannot be updated")
	})

	t.Run("unchanged reference cannot produce a patch", func(t *testing.T) {
		_, err := existing.AsPatchUpdateSubscriptionReference(existing)
		require.ErrorContains(t, err, "at least one subscription reference update is required")
	})
}
