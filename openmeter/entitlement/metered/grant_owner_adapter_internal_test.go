package meteredentitlement

import (
	"encoding/json"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func mustMarshalUnitConfig(t *testing.T, uc productcatalog.UnitConfig) *string {
	t.Helper()
	b, err := json.Marshal(uc)
	require.NoError(t, err)
	return lo.ToPtr(string(b))
}

// TestBuildUsageConverter covers the OM-400 conversion at the credit boundary in
// isolation: the flag gate, the nil/identity paths that keep balances byte-identical
// when off, the divide/multiply conversions (no rounding), and fail-closed parsing.
func TestBuildUsageConverter(t *testing.T) {
	divideByGB := mustMarshalUnitConfig(t, productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1_000_000_000),
	})
	// A ceiling rounding is set to prove the converter uses the unrounded value.
	divideWithRounding := mustMarshalUnitConfig(t, productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1_000_000_000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	})
	multiplyBy2 := mustMarshalUnitConfig(t, productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationMultiply,
		ConversionFactor: alpacadecimal.NewFromInt(2),
	})
	zeroFactor := mustMarshalUnitConfig(t, productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(0),
	})

	t.Run("flag off returns identity even when a config is present", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: false}
		conv, err := e.buildUsageConverter(divideByGB)
		require.NoError(t, err)
		assert.Nil(t, conv)
	})

	t.Run("nil config returns identity", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		conv, err := e.buildUsageConverter(nil)
		require.NoError(t, err)
		assert.Nil(t, conv)
	})

	t.Run("divide converts without rounding", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		conv, err := e.buildUsageConverter(divideByGB)
		require.NoError(t, err)
		require.NotNil(t, conv)
		assert.InDelta(t, 99.3, conv(99_300_000_000), 1e-6)
	})

	t.Run("rounding mode does not affect the converted balance value", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		conv, err := e.buildUsageConverter(divideWithRounding)
		require.NoError(t, err)
		require.NotNil(t, conv)
		// ceiling would give 100; balance checks must see the precise 99.3.
		assert.InDelta(t, 99.3, conv(99_300_000_000), 1e-6)
	})

	t.Run("multiply converts", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		conv, err := e.buildUsageConverter(multiplyBy2)
		require.NoError(t, err)
		require.NotNil(t, conv)
		assert.InDelta(t, 10.0, conv(5), 1e-9)
	})

	t.Run("malformed JSON fails closed", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		_, err := e.buildUsageConverter(lo.ToPtr("not json"))
		require.Error(t, err)
	})

	t.Run("invalid config fails closed", func(t *testing.T) {
		e := &entitlementGrantOwner{unitConfigEnabled: true}
		_, err := e.buildUsageConverter(zeroFactor)
		require.Error(t, err)
	})
}
