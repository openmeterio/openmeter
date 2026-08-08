package meta

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestPatchClearOverride(t *testing.T) {
	t.Run("given an API request patch when validating it then it is valid", func(t *testing.T) {
		patch, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
		})

		require.NoError(t, err)
		require.NoError(t, patch.Validate())
		require.Equal(t, PatchTypeClearOverride, patch.Op())
		require.Equal(t, TriggerClearOverride, patch.Trigger())
	})

	t.Run("given a clear override patch when selecting its target then it selects the override layer", func(t *testing.T) {
		patch, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceAPIRequest,
		})
		require.NoError(t, err)

		target, err := patch.GetTargetLayer(layeredIntentReaderForTest{})
		require.NoError(t, err)
		require.Equal(t, ChangeTargetOverride, target)
	})

	t.Run("given a system patch when creating a clear override patch then it is rejected", func(t *testing.T) {
		_, err := NewPatchClearOverride(NewPatchClearOverrideInput{
			ChangeSource: billing.ChangeSourceSystem,
		})

		require.ErrorContains(t, err, "must be api_request")
	})

	t.Run("given a malformed clear override patch when selecting its target then it is rejected", func(t *testing.T) {
		_, err := (PatchClearOverride{changeSource: billing.ChangeSourceSystem}).GetTargetLayer(layeredIntentReaderForTest{})

		require.ErrorContains(t, err, "must be api_request")
	})
}
