package feature

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/assert"
)

func TestUpdateFeatureInputsValidate(t *testing.T) {
	validUnitCost := nullable.NewNullableWithValue(UnitCost{
		Type: UnitCostTypeManual,
		Manual: &ManualUnitCost{
			Amount: alpacadecimal.NewFromFloat(0.05),
		},
	})

	t.Run("valid with unit cost", func(t *testing.T) {
		input := UpdateFeatureInputs{
			Namespace: "ns",
			ID:        "feat-1",
			UnitCost:  validUnitCost,
		}
		assert.NoError(t, input.Validate())
	})

	t.Run("valid with clear unit cost (null)", func(t *testing.T) {
		input := UpdateFeatureInputs{
			Namespace: "ns",
			ID:        "feat-1",
			UnitCost:  nullable.NewNullNullable[UnitCost](),
		}
		assert.NoError(t, input.Validate())
	})

	t.Run("invalid without unit cost specified", func(t *testing.T) {
		input := UpdateFeatureInputs{
			Namespace: "ns",
			ID:        "feat-1",
		}
		err := input.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unitCost is required")
	})

	t.Run("invalid missing namespace", func(t *testing.T) {
		input := UpdateFeatureInputs{
			ID:       "feat-1",
			UnitCost: validUnitCost,
		}
		err := input.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("invalid missing id", func(t *testing.T) {
		input := UpdateFeatureInputs{
			Namespace: "ns",
			UnitCost:  validUnitCost,
		}
		err := input.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})
}
