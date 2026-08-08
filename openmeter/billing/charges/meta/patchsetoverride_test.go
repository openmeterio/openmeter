package meta

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ Patch = PatchSetOverride[setOverrideMutableFieldsForTest]{}

type setOverrideMutableFieldsForTest struct {
	Name      string
	DeletedAt *time.Time
	Invalid   bool
}

func (f setOverrideMutableFieldsForTest) Clone() setOverrideMutableFieldsForTest {
	return f
}

func (f setOverrideMutableFieldsForTest) GetIntentDeletedAt() *time.Time {
	return f.DeletedAt
}

func (f setOverrideMutableFieldsForTest) Validate() error {
	if f.Invalid {
		return errors.New("invalid mutable fields")
	}

	return nil
}

func TestPatchSetOverride(t *testing.T) {
	fields := setOverrideMutableFieldsForTest{Name: "override"}

	t.Run("given an API override patch when the input is mutated afterwards then it retains a snapshot", func(t *testing.T) {
		inputFields := fields.Clone()
		patch, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: inputFields,
		})
		require.NoError(t, err)

		inputFields.Name = "mutated"

		require.Equal(t, "override", patch.GetIntentMutableFields().Name)
	})

	t.Run("given an override patch when selecting its target then it always uses the override layer", func(t *testing.T) {
		patch, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: fields,
		})
		require.NoError(t, err)

		target, err := patch.GetTargetLayer(layeredIntentReaderForTest{})
		require.NoError(t, err)
		require.Equal(t, ChangeTargetOverride, target)
	})

	t.Run("given a non-API source when creating an override patch then it is rejected", func(t *testing.T) {
		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceSystem,
			IntentMutableFields: fields,
		})

		require.ErrorContains(t, err, "must be api_request")
	})

	t.Run("given an invalid override patch when selecting its target then it is rejected", func(t *testing.T) {
		patch := PatchSetOverride[setOverrideMutableFieldsForTest]{changeSource: billing.ChangeSourceSystem}

		_, err := patch.GetTargetLayer(layeredIntentReaderForTest{})

		require.ErrorContains(t, err, "must be api_request")
	})

	t.Run("given a deleted override intent when creating an override patch then it is rejected", func(t *testing.T) {
		deletedAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
		deletedFields := fields.Clone()
		deletedFields.DeletedAt = &deletedAt

		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: deletedFields,
		})

		require.ErrorContains(t, err, "intent deleted at")
	})

	t.Run("given invalid mutable fields when creating an override patch then it is rejected", func(t *testing.T) {
		invalidFields := fields.Clone()
		invalidFields.Invalid = true

		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: invalidFields,
		})

		require.ErrorContains(t, err, "invalid mutable fields")
	})
}
