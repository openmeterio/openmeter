package productcatalog

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestUnitConfigApply(t *testing.T) {
	t.Run("nil receiver is identity", func(t *testing.T) {
		var c *UnitConfig
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.True(t, converted.Equal(raw))
		assert.True(t, invoiced.Equal(raw))
	})

	t.Run("multiply without rounding produces precise converted", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationMultiply,
			ConversionFactor: decimal.NewFromFloat(1.5),
		}
		raw := decimal.NewFromFloat(4.2)

		converted, invoiced := c.Apply(raw)
		assert.True(t, converted.Equal(decimal.NewFromFloat(6.3)), "converted: %s", converted.String())
		assert.True(t, invoiced.Equal(converted), "no rounding → invoiced equals converted")
	})

	t.Run("divide without rounding produces precise converted", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
		}
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.True(t, converted.Equal(decimal.NewFromFloat(1.247)), "converted: %s", converted.String())
		assert.True(t, invoiced.Equal(converted))
	})

	t.Run("divide with ceiling rounding to whole numbers (package-style)", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeCeiling),
		}
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.True(t, converted.Equal(decimal.NewFromFloat(1.247)), "converted stays precise: %s", converted.String())
		assert.True(t, invoiced.Equal(decimal.NewFromInt(2)), "invoiced rounds up: %s", invoiced.String())
	})

	t.Run("divide with floor rounding", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeFloor),
		}
		raw := decimal.NewFromInt(1999)

		_, invoiced := c.Apply(raw)
		assert.True(t, invoiced.Equal(decimal.NewFromInt(1)), "invoiced floors: %s", invoiced.String())
	})

	t.Run("divide with half_up rounding", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeHalfUp),
		}

		_, invoicedUp := c.Apply(decimal.NewFromInt(1500))
		assert.True(t, invoicedUp.Equal(decimal.NewFromInt(2)), "1.500 rounds up: %s", invoicedUp.String())

		_, invoicedDown := c.Apply(decimal.NewFromInt(1499))
		assert.True(t, invoicedDown.Equal(decimal.NewFromInt(1)), "1.499 rounds down: %s", invoicedDown.String())
	})

	t.Run("explicit precision retains decimal places", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeCeiling),
			Precision:        lo.ToPtr(2),
		}
		raw := decimal.NewFromInt(1247)

		_, invoiced := c.Apply(raw)
		assert.True(t, invoiced.Equal(decimal.NewFromFloat(1.25)), "1.247 ceil to 2dp: %s", invoiced.String())
	})

	t.Run("rounding mode none leaves converted intact", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeNone),
		}
		raw := decimal.NewFromInt(1247)

		converted, invoiced := c.Apply(raw)
		assert.True(t, invoiced.Equal(converted))
	})

	t.Run("v1 dynamic equivalence: multiply", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationMultiply,
			ConversionFactor: decimal.NewFromFloat(1.5),
		}
		raw := decimal.NewFromFloat(4.20)

		_, invoiced := c.Apply(raw)
		expected := raw.Mul(decimal.NewFromFloat(1.5))
		assert.True(t, invoiced.Equal(expected), "Dynamic{1.5} parity: %s vs %s", invoiced.String(), expected.String())
	})

	t.Run("v1 package equivalence: divide + ceiling", func(t *testing.T) {
		c := &UnitConfig{
			Operation:        UnitConfigOperationDivide,
			ConversionFactor: decimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(UnitConfigRoundingModeCeiling),
		}

		_, exact := c.Apply(decimal.NewFromInt(1000))
		assert.True(t, exact.Equal(decimal.NewFromInt(1)), "exact boundary: %s", exact.String())

		_, partial := c.Apply(decimal.NewFromInt(1247))
		assert.True(t, partial.Equal(decimal.NewFromInt(2)), "partial: %s", partial.String())

		_, none := c.Apply(decimal.NewFromInt(0))
		assert.True(t, none.Equal(decimal.NewFromInt(0)), "zero: %s", none.String())
	})
}
