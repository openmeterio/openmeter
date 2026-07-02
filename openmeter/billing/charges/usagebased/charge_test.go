package usagebased

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// TestOverridableIntentGetEffectiveUnitConfig locks in that the override layer is a
// full snapshot of the effective mutable fields: an override created from the effective
// intent inherits the base unit_config, so an unrelated edit does not drop the
// conversion, while an explicit nil is a genuine cleared state.
func TestOverridableIntentGetEffectiveUnitConfig(t *testing.T) {
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	base := Intent{
		IntentMutableFields: IntentMutableFields{
			Price:      *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			UnitConfig: unitConfig,
		},
		FeatureKey: "f",
	}

	oi := base.AsOverridableIntent()

	t.Run("override with an unrelated edit inherits the base unit_config", func(t *testing.T) {
		// Mirror how the state machine creates the first override: snapshot the full
		// effective mutable fields, then change one unrelated field only.
		overrideFields := oi.GetEffectiveIntent().IntentMutableFields
		overrideFields.Name = "unrelated override edit"

		withOverride := NewOverridableIntent(base, &overrideFields)

		got := withOverride.GetEffectiveUnitConfig()
		require.NotNil(t, got, "unrelated override must not drop the base unit_config")
		require.True(t, unitConfig.Equal(got))
	})

	t.Run("explicit nil on the override is a cleared state, not inherited", func(t *testing.T) {
		clearedFields := oi.GetEffectiveIntent().IntentMutableFields
		clearedFields.UnitConfig = nil

		cleared := NewOverridableIntent(base, &clearedFields)
		require.Nil(t, cleared.GetEffectiveUnitConfig())
	})
}
