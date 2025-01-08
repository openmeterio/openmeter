package productcatalog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripeTaxConfig(t *testing.T) {
	tests := []struct {
		Name          string
		TaxConfig     StripeTaxConfig
		ExpectedError error
	}{
		{
			Name: "valid",
			TaxConfig: StripeTaxConfig{
				Code: "txcd_99999999",
			},
			ExpectedError: nil,
		},
		{
			Name: "invalid",
			TaxConfig: StripeTaxConfig{
				Code: "invalid_tax_code",
			},
			ExpectedError: errors.New("invalid product tax code: invalid_tax_code"),
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
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			match := test.Left.Equal(test.Right)
			assert.Equal(t, test.ExpectedResult, match)
		})
	}
}
