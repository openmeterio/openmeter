package grant_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/unitconfig"
)

// TestOwnerConvertUsage covers the OM-400 conversion at the credit boundary: identity
// when the owner has no UnitConfig, and conversion without rounding (balance checks
// always use the precise value) when it does.
func TestOwnerConvertUsage(t *testing.T) {
	t.Run("no unit config is identity", func(t *testing.T) {
		owner := grant.Owner{}
		assert.Equal(t, 99_300_000_000.0, owner.ConvertUsage(99_300_000_000))
	})

	t.Run("divide converts", func(t *testing.T) {
		owner := grant.Owner{
			UnitConfig: &unitconfig.UnitConfig{
				Operation:        unitconfig.UnitConfigOperationDivide,
				ConversionFactor: alpacadecimal.NewFromInt(1_000_000_000),
			},
		}
		assert.InDelta(t, 99.3, owner.ConvertUsage(99_300_000_000), 1e-6)
	})

	t.Run("rounding is ignored — balance uses the precise converted value", func(t *testing.T) {
		owner := grant.Owner{
			UnitConfig: &unitconfig.UnitConfig{
				Operation:        unitconfig.UnitConfigOperationDivide,
				ConversionFactor: alpacadecimal.NewFromInt(1_000_000_000),
				Rounding:         unitconfig.UnitConfigRoundingModeCeiling,
			},
		}
		// ceiling would invoice 100; the balance check must see 99.3.
		assert.InDelta(t, 99.3, owner.ConvertUsage(99_300_000_000), 1e-6)
	})
}
