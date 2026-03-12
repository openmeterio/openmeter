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
			Name: "tax code id valid",
			TaxConfig: TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			ExpectedError: nil,
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
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				TaxCodeId: nil,
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: nil,
			Right: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
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
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
		},
		{
			Name: "Right nil",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: nil,
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
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
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				Behavior:  lo.ToPtr(ExclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(ExclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
			},
		},
		{
			Name: "Right overrides left partially - behavior",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				Behavior: lo.ToPtr(ExclusiveTaxBehavior),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(ExclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
		},
		{
			Name: "Right overrides left partially - tax code id",
			Left: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			},
			Right: &TaxConfig{
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
			},
			Expected: &TaxConfig{
				Behavior:  lo.ToPtr(InclusiveTaxBehavior),
				TaxCodeId: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB0"),
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
