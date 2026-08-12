package meta

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestPatchClearOverride(t *testing.T) {
	t.Run("targets the override layer", func(t *testing.T) {
		patch, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
		})
		require.NoError(t, err)

		target, err := patch.GetTargetLayer(layeredIntentReaderForTest{
			baseManagedBy: billing.ManuallyManagedLine,
		})
		require.NoError(t, err)
		require.Equal(t, ChangeTargetOverride, target)
	})

	t.Run("rejects missing intent when selecting target", func(t *testing.T) {
		patch, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
		})
		require.NoError(t, err)

		_, err = patch.GetTargetLayer(nil)
		require.ErrorContains(t, err, "intent is required")
	})

	t.Run("rejects non API source", func(t *testing.T) {
		_, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceSystem,
		})

		require.ErrorContains(t, err, "must be api_request")
	})
}
