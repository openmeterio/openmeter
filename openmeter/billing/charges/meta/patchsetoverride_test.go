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

	t.Run("snapshots mutable fields", func(t *testing.T) {
		inputFields := fields.Clone()
		patch, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: inputFields,
		})
		require.NoError(t, err)

		inputFields.Name = "mutated input"
		returnedFields := patch.GetIntentMutableFields()
		returnedFields.Name = "mutated getter result"

		require.Equal(t, "override", patch.GetIntentMutableFields().Name)
	})

	t.Run("always targets the override layer", func(t *testing.T) {
		patch, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: fields,
		})
		require.NoError(t, err)

		target, err := patch.GetTargetLayer(layeredIntentReaderForTest{
			baseManagedBy: billing.ManuallyManagedLine,
		})
		require.NoError(t, err)
		require.Equal(t, ChangeTargetOverride, target)
	})

	t.Run("rejects missing intent when selecting target", func(t *testing.T) {
		patch, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: fields,
		})
		require.NoError(t, err)

		_, err = patch.GetTargetLayer(nil)
		require.ErrorContains(t, err, "intent is required")
	})

	t.Run("rejects non API source", func(t *testing.T) {
		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceSystem,
			IntentMutableFields: fields,
		})

		require.ErrorContains(t, err, "must be api_request")
	})

	t.Run("rejects deleted override intent", func(t *testing.T) {
		deletedAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
		deletedFields := fields.Clone()
		deletedFields.DeletedAt = &deletedAt

		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: deletedFields,
		})

		require.ErrorContains(t, err, "intent deleted at")
	})

	t.Run("rejects invalid mutable fields", func(t *testing.T) {
		invalidFields := fields.Clone()
		invalidFields.Invalid = true

		_, err := NewPatchSetOverride(NewPatchSetOverrideInput[setOverrideMutableFieldsForTest]{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: invalidFields,
		})

		require.ErrorContains(t, err, "invalid mutable fields")
	})
}
