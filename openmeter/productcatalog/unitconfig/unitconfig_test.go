package unitconfig

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitConfigApply(t *testing.T) {
	t.Run("nil receiver is identity", func(t *testing.T) {
		var c *UnitConfig
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.Equal(t, raw.InexactFloat64(), converted.InexactFloat64())
		assert.Equal(t, raw.InexactFloat64(), invoiced.InexactFloat64())
	})

	t.Run("unknown operation is identity (never panics mid-billing)", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        "bogus",
			ConversionFactor: decimal.NewFromInt(1000),
		}
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.Equal(t, float64(1247), converted.InexactFloat64())
		assert.Equal(t, float64(1247), invoiced.InexactFloat64())
	})

	t.Run("multiply without rounding produces precise converted", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationMultiply,
			ConversionFactor: decimal.NewFromFloat(1.5),
		}

		converted, invoiced := c.Apply(decimal.NewFromFloat(4.2))
		assert.Equal(t, 6.3, converted.InexactFloat64())
		assert.Equal(t, converted.InexactFloat64(), invoiced.InexactFloat64(), "no rounding → invoiced equals converted")
	})

	t.Run("divide without rounding produces precise converted", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
		}

		converted, invoiced := c.Apply(decimal.NewFromInt(1247))
		assert.Equal(t, 1.247, converted.InexactFloat64())
		assert.Equal(t, converted.InexactFloat64(), invoiced.InexactFloat64())
	})

	t.Run("ceiling rounds up, converted stays precise (package-style)", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeCeiling,
		}

		converted, invoiced := c.Apply(decimal.NewFromInt(1247))
		assert.Equal(t, 1.247, converted.InexactFloat64(), "converted stays precise")
		assert.Equal(t, float64(2), invoiced.InexactFloat64(), "invoiced rounds up")

		// exact boundary and zero
		_, exact := c.Apply(decimal.NewFromInt(1000))
		assert.Equal(t, float64(1), exact.InexactFloat64())

		_, zero := c.Apply(decimal.NewFromInt(0))
		assert.Equal(t, float64(0), zero.InexactFloat64())
	})

	t.Run("floor rounds down (incomplete unit)", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeFloor,
		}

		_, invoiced := c.Apply(decimal.NewFromInt(1999))
		assert.Equal(t, float64(1), invoiced.InexactFloat64())
	})

	t.Run("half_up rounds to nearest, ties away from zero", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeHalfUp,
		}

		// 2.4 → 2, 2.5 → 3 (tie up), 2.6 → 3, 3.5 → 4 (tie up)
		_, down := c.Apply(decimal.NewFromInt(2400))
		assert.Equal(t, float64(2), down.InexactFloat64())

		_, tie := c.Apply(decimal.NewFromInt(2500))
		assert.Equal(t, float64(3), tie.InexactFloat64(), "exact tie rounds away from zero")

		_, up := c.Apply(decimal.NewFromInt(2600))
		assert.Equal(t, float64(3), up.InexactFloat64())

		_, tie2 := c.Apply(decimal.NewFromInt(3500))
		assert.Equal(t, float64(4), tie2.InexactFloat64())
	})

	// Only negatives prove "away from zero" rather than "toward +∞"; spec matches
	// Java RoundingMode.HALF_UP / Python ROUND_HALF_UP.
	t.Run("half_up tie is away from zero for negatives", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeHalfUp,
		}

		_, invoiced := c.Apply(decimal.NewFromInt(-2500))
		assert.Equal(t, float64(-3), invoiced.InexactFloat64(), "−2.5 → −3, not −2")
	})

	t.Run("precision moves where the tie is evaluated", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(100),
			Rounding:         UnitConfigRoundingModeHalfUp,
			Precision:        1,
		}

		// 245/100 = 2.45 → 2.5 (tie at 2nd dp → up)
		_, tie := c.Apply(decimal.NewFromInt(245))
		assert.Equal(t, 2.5, tie.InexactFloat64())

		// 244/100 = 2.44 → 2.4
		_, near := c.Apply(decimal.NewFromInt(244))
		assert.Equal(t, 2.4, near.InexactFloat64())
	})

	t.Run("ceiling with precision retains decimal places", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeCeiling,
			Precision:        2,
		}

		_, invoiced := c.Apply(decimal.NewFromInt(1247))
		assert.Equal(t, 1.25, invoiced.InexactFloat64(), "1.247 ceil to 2dp")
	})

	t.Run("rounding none and empty leave converted intact", func(t *testing.T) {
		for _, mode := range []UnitConfigRoundingMode{UnitConfigRoundingModeNone, ""} {
			c := &UnitConfig{
				Operation:        UnitConfigOperationDivide,
				ConversionFactor: decimal.NewFromInt(1000),
				Rounding:         mode,
			}

			converted, invoiced := c.Apply(decimal.NewFromInt(1247))
			assert.Equal(t, converted.InexactFloat64(), invoiced.InexactFloat64(), "mode %q", mode)
		}
	})
}

func TestUnitConfigValidate(t *testing.T) {
	t.Run("nil is valid", func(t *testing.T) {
		var c *UnitConfig
		require.NoError(t, c.Validate())
	})

	t.Run("valid config", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeCeiling,
			Precision:        2,
		}
		require.NoError(t, c.Validate())
	})

	t.Run("valid with defaults (empty rounding, zero precision)", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationMultiply,
			ConversionFactor: decimal.NewFromFloat(1.5),
		}
		require.NoError(t, c.Validate())
	})

	t.Run("invalid operation", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        "bogus",
			ConversionFactor: decimal.NewFromInt(1000),
		}
		require.Error(t, c.Validate())
	})

	t.Run("conversion_factor must be greater than zero", func(t *testing.T) {
		zero := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(0),
		}
		require.Error(t, zero.Validate())

		negative := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(-5),
		}
		require.Error(t, negative.Validate())
	})

	t.Run("invalid rounding mode", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         "bogus",
		}
		require.Error(t, c.Validate())
	})

	t.Run("negative precision is rejected when rounding is active", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeCeiling,
			Precision:        -1,
		}
		require.Error(t, c.Validate())
	})

	t.Run("negative precision is ignored when rounding is none", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeNone,
			Precision:        -1,
		}
		require.NoError(t, c.Validate())
	})
}

func TestUnitConfigEqual(t *testing.T) {
	base := func() *UnitConfig {
		return &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         UnitConfigRoundingModeCeiling,
			Precision:        2,
			DisplayUnit:      lo.ToPtr("GB"),
		}
	}

	t.Run("both nil", func(t *testing.T) {
		var a, b *UnitConfig
		assert.True(t, a.Equal(b))
	})

	t.Run("one nil", func(t *testing.T) {
		assert.False(t, base().Equal(nil))
		assert.False(t, (*UnitConfig)(nil).Equal(base()))
	})

	t.Run("equal", func(t *testing.T) {
		assert.True(t, base().Equal(base()))
	})

	t.Run("empty rounding equals explicit none", func(t *testing.T) {
		a := &UnitConfig{Operation: UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000)}
		b := &UnitConfig{Operation: UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000), Rounding: UnitConfigRoundingModeNone}
		assert.True(t, a.Equal(b))
	})

	t.Run("precision is inert when rounding is none", func(t *testing.T) {
		// Apply and Validate ignore Precision when rounding is none, so differing
		// precision must not make two behaviorally identical configs unequal.
		a := &UnitConfig{Operation: UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000), Rounding: UnitConfigRoundingModeNone, Precision: 0}
		b := &UnitConfig{Operation: UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000), Rounding: UnitConfigRoundingModeNone, Precision: 5}
		assert.True(t, a.Equal(b))
	})

	t.Run("differs by each field", func(t *testing.T) {
		differ := base()
		differ.Operation = UnitConfigOperationMultiply
		assert.False(t, base().Equal(differ))

		differ = base()
		differ.ConversionFactor = decimal.NewFromInt(2000)
		assert.False(t, base().Equal(differ))

		differ = base()
		differ.Rounding = UnitConfigRoundingModeFloor
		assert.False(t, base().Equal(differ))

		differ = base()
		differ.Precision = 3
		assert.False(t, base().Equal(differ))

		differ = base()
		differ.DisplayUnit = lo.ToPtr("MB")
		assert.False(t, base().Equal(differ))
	})
}

func TestUnitConfigClone(t *testing.T) {
	original := &UnitConfig{
		Operation:        UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         UnitConfigRoundingModeCeiling,
		Precision:        2,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	clone := original.Clone()
	assert.True(t, original.Equal(&clone))

	// Mutating the clone's DisplayUnit must not affect the original (deep copy).
	*clone.DisplayUnit = "MB"
	assert.Equal(t, "GB", *original.DisplayUnit)

	clone.DisplayUnit = nil
	assert.NotNil(t, original.DisplayUnit)
}
