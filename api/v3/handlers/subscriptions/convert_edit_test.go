package subscriptions

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
)

// editOperation wraps a concrete edit operation model into the discriminated
// union the handler decodes, so each subtest reads as the wire payload it maps.
func editOperation(t *testing.T, set func(op *apiv3models.BillingSubscriptionEditOperation) error) apiv3models.BillingSubscriptionEditOperation {
	t.Helper()

	var op apiv3models.BillingSubscriptionEditOperation
	require.NoError(t, set(&op))

	return op
}

func TestFromAPIBillingSubscriptionEditOperation(t *testing.T) {
	t.Run("add_item maps to PatchAddItem with the rate card key", func(t *testing.T) {
		// given: an add_item operation carrying a free flat-fee rate card
		var price apiv3models.BillingPrice
		require.NoError(t, price.FromBillingPriceFree(apiv3models.BillingPriceFree{Type: "free"}))

		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditAddItem(apiv3models.BillingSubscriptionEditAddItem{
				Type:     apiv3models.BillingSubscriptionEditAddItemTypeAddItem,
				PhaseKey: "trial",
				RateCard: apiv3models.BillingRateCard{Key: "setup", Name: "Setup", Price: price},
			})
		})

		// when: it is mapped to a domain patch
		p, err := FromAPIBillingSubscriptionEditOperation(op)

		// then: the patch is an add-item targeting the phase, with the item key
		// derived from the rate card
		require.NoError(t, err)
		add, ok := p.(patch.PatchAddItem)
		require.True(t, ok, "expected PatchAddItem, got %T", p)
		require.Equal(t, "trial", add.PhaseKey)
		require.Equal(t, "setup", add.ItemKey)
		require.Equal(t, subscription.PatchOperationAdd, add.Op())
	})

	t.Run("remove_item maps to PatchRemoveItem", func(t *testing.T) {
		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditRemoveItem(apiv3models.BillingSubscriptionEditRemoveItem{
				Type:     apiv3models.BillingSubscriptionEditRemoveItemTypeRemoveItem,
				PhaseKey: "trial",
				ItemKey:  "setup",
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		require.NoError(t, err)
		rm, ok := p.(patch.PatchRemoveItem)
		require.True(t, ok, "expected PatchRemoveItem, got %T", p)
		require.Equal(t, "trial", rm.PhaseKey)
		require.Equal(t, "setup", rm.ItemKey)
		require.Equal(t, subscription.PatchOperationRemove, rm.Op())
	})

	t.Run("add_phase maps to PatchAddPhase and parses durations", func(t *testing.T) {
		// given: a new phase with both start_after and duration set
		startAfter := apiv3models.ISO8601Duration("P1M")
		duration := apiv3models.ISO8601Duration("P2M")
		description := "Second phase"

		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditAddPhase(apiv3models.BillingSubscriptionEditAddPhase{
				Type: apiv3models.BillingSubscriptionEditAddPhaseTypeAddPhase,
				Phase: apiv3models.BillingSubscriptionPhaseCreate{
					Key:         "second",
					Name:        "Second",
					Description: &description,
					StartAfter:  nullable.NewNullableWithValue(startAfter),
					Duration:    &duration,
				},
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		// then: the phase-create input carries the parsed durations and metadata
		require.NoError(t, err)
		add, ok := p.(patch.PatchAddPhase)
		require.True(t, ok, "expected PatchAddPhase, got %T", p)
		require.Equal(t, "second", add.PhaseKey)
		require.Equal(t, "second", add.CreateInput.PhaseKey)
		require.Equal(t, "Second", add.CreateInput.Name)
		require.Equal(t, "P1M", add.CreateInput.StartAfter.ISOString().String())
		require.NotNil(t, add.CreateInput.Duration)
		require.Equal(t, "P2M", add.CreateInput.Duration.ISOString().String())
		require.Equal(t, subscription.PatchOperationAdd, add.Op())
	})

	t.Run("add_phase with null start_after starts immediately", func(t *testing.T) {
		// given: a phase with an explicit null start_after (subscription start) and
		// duration omitted (last phase). start_after is required-but-nullable, so null
		// is the wire form of "at subscription start", matching v1.
		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditAddPhase(apiv3models.BillingSubscriptionEditAddPhase{
				Type: apiv3models.BillingSubscriptionEditAddPhaseTypeAddPhase,
				Phase: apiv3models.BillingSubscriptionPhaseCreate{
					Key:        "final",
					Name:       "Final",
					StartAfter: nullable.NewNullNullable[apiv3models.ISO8601Duration](),
				},
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		// then: start_after resolves to the zero (subscription-start) offset and duration is nil
		require.NoError(t, err)
		add, ok := p.(patch.PatchAddPhase)
		require.True(t, ok, "expected PatchAddPhase, got %T", p)
		require.True(t, add.CreateInput.StartAfter.IsZero())
		require.Nil(t, add.CreateInput.Duration)
		require.Equal(t, subscription.PatchOperationAdd, add.Op())
	})

	t.Run("remove_phase maps the shift direction", func(t *testing.T) {
		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditRemovePhase(apiv3models.BillingSubscriptionEditRemovePhase{
				Type:     apiv3models.BillingSubscriptionEditRemovePhaseTypeRemovePhase,
				PhaseKey: "trial",
				Shift:    apiv3models.BillingSubscriptionRemovePhaseShiftingNext,
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		require.NoError(t, err)
		rm, ok := p.(patch.PatchRemovePhase)
		require.True(t, ok, "expected PatchRemovePhase, got %T", p)
		require.Equal(t, "trial", rm.PhaseKey)
		require.Equal(t, subscription.RemoveSubscriptionPhaseShiftNext, rm.RemoveInput.Shift)
		require.Equal(t, subscription.PatchOperationRemove, rm.Op())
	})

	t.Run("stretch_phase parses the extend_by duration", func(t *testing.T) {
		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditStretchPhase(apiv3models.BillingSubscriptionEditStretchPhase{
				Type:     apiv3models.BillingSubscriptionEditStretchPhaseTypeStretchPhase,
				PhaseKey: "trial",
				ExtendBy: apiv3models.ISO8601Duration("P1W"),
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		require.NoError(t, err)
		st, ok := p.(patch.PatchStretchPhase)
		require.True(t, ok, "expected PatchStretchPhase, got %T", p)
		require.Equal(t, "trial", st.PhaseKey)
		require.Equal(t, "P1W", st.Duration.ISOString().String())
		require.Equal(t, subscription.PatchOperationStretch, st.Op())
	})

	t.Run("unschedule_edit maps to PatchUnscheduleEdit", func(t *testing.T) {
		op := editOperation(t, func(op *apiv3models.BillingSubscriptionEditOperation) error {
			return op.FromBillingSubscriptionEditUnscheduleEdit(apiv3models.BillingSubscriptionEditUnscheduleEdit{
				Type: apiv3models.BillingSubscriptionEditUnscheduleEditTypeUnscheduleEdit,
			})
		})

		p, err := FromAPIBillingSubscriptionEditOperation(op)

		require.NoError(t, err)
		un, ok := p.(patch.PatchUnscheduleEdit)
		require.True(t, ok, "expected PatchUnscheduleEdit, got %T", p)
		require.Equal(t, subscription.PatchOperationUnschedule, un.Op())
	})
}
