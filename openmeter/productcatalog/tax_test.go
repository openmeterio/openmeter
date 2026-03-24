package productcatalog

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestTaxConfigValidation(t *testing.T) {
	tests := []struct {
		Name          string
		TaxConfig     TaxConfig
		ExpectedError error
	}{
		{
			Name: "stripe valid",
			TaxConfig: TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedError: nil,
		},
		{
			Name: "stripe invalid",
			TaxConfig: TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "invalid_tax_code",
				},
			},
			ExpectedError: errors.New("validation error: invalid stripe config: validation error: invalid product tax code: invalid_tax_code"),
		},
		{
			Name: "behavior valid",
			TaxConfig: TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedError: nil,
		},
		{
			Name: "behavior invalid",
			TaxConfig: TaxConfig{
				Behavior: (*TaxBehavior)(lo.ToPtr("invalid_behavior")),
			},
			ExpectedError: errors.New("validation error: invalid tax behavior: invalid_behavior"),
		},
	}

	for _, test := range tests {
		err := test.TaxConfig.Validate()
		if test.ExpectedError == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, test.ExpectedError.Error())
		}
	}
}

func TestTaxConfigEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  *TaxConfig
		Right *TaxConfig

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: nil,
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: nil,
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			ExpectedResult: false,
		},
		{
			Name: "Equal - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right:          &TaxConfig{},
			ExpectedResult: false,
		},
		{
			Name: "Right diff - behavior",
			Left: nil,
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			ExpectedResult: false,
		},
		{
			Name: "Equal - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right:          &TaxConfig{},
			ExpectedResult: false,
		},
		{
			Name: "Right diff - TaxCodeID",
			Left: nil,
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff - TaxCodeID",
			Left: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
			},
			Right: &TaxConfig{
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			match := test.Left.Equal(test.Right)
			assert.Equal(t, test.ExpectedResult, match)
		})
	}
}

func TestMergeTaxConfigs(t *testing.T) {
	tests := []struct {
		Name     string
		Left     *TaxConfig
		Right    *TaxConfig
		Expected *TaxConfig
	}{
		{
			Name: "Left nil",
			Left: nil,
			Right: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right nil",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: nil,
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name:     "Left and Right nil",
			Left:     nil,
			Right:    nil,
			Expected: nil,
		},
		{
			Name: "Right overrides left fully",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
		{
			Name: "Right overrides left partially - behavior",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right overrides left partially - stripe",
			Left: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
			Expected: &TaxConfig{
				Behavior: lo.ToPtr(InclusiveTaxBehavior),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
		{
			Name: "Right overrides left partially - TaxCodeID",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
		},
		{
			Name: "Right overrides left partially - TaxCodeID and Stripe",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999999",
				},
			},
			Right: &TaxConfig{
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV4"),
				Stripe: &StripeTaxConfig{
					Code: "txcd_99999998",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			merged := MergeTaxConfigs(test.Left, test.Right)
			assert.Equal(t, test.Expected, merged)
		})
	}
}

func TestTaxConfigClone(t *testing.T) {
	original := TaxConfig{
		Behavior:  lo.ToPtr(InclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("01AN4Z07BY79KA1307SR9X4MV3"),
		Stripe: &StripeTaxConfig{
			Code: "txcd_99999999",
		},
	}

	cloned := original.Clone()

	// Cloned value must be equal to the original
	assert.True(t, original.Equal(&cloned))

	// Mutate the clone's pointer fields
	*cloned.TaxCodeID = "01AN4Z07BY79KA1307SR9X4MV4"
	*cloned.Behavior = ExclusiveTaxBehavior
	cloned.Stripe.Code = "txcd_00000000"

	// Original must be unchanged
	assert.Equal(t, "01AN4Z07BY79KA1307SR9X4MV3", *original.TaxCodeID)
	assert.Equal(t, InclusiveTaxBehavior, *original.Behavior)
	assert.Equal(t, "txcd_99999999", original.Stripe.Code)

	// Pointers must not be shared
	assert.NotSame(t, original.TaxCodeID, cloned.TaxCodeID)
	assert.NotSame(t, original.Behavior, cloned.Behavior)
	assert.NotSame(t, original.Stripe, cloned.Stripe)

	// Values must now differ
	assert.False(t, original.Equal(&cloned))
}
