package productcatalog

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDiscount_JSON(t *testing.T) {
	tests := []struct {
		Name          string
		Discounts     Discounts
		ExpectedError bool
	}{
		{
			Name: "Valid - percentage",
			Discounts: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(99.9),
				},
			},
		},
		{
			Name: "Valid - usage",
			Discounts: Discounts{
				Usage: &UsageDiscount{
					Quantity: decimal.NewFromInt(100),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.Discounts)
			require.NoError(t, err)

			t.Logf("Serialized Discount: %s", string(b))

			d := Discounts{}
			err = json.Unmarshal(b, &d)
			require.NoError(t, err)

			assert.Equal(t, test.Discounts, d)
		})
	}
}

func TestDiscountsEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  Discounts
		Right Discounts

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(100),
				},
			},
			Right: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(100),
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "Diff",
			Left: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(100),
				},
			},
			Right: Discounts{
				Usage: &UsageDiscount{
					Quantity: decimal.NewFromInt(100),
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

func TestDiscountsValidateForPrice(t *testing.T) {
	tests := []struct {
		Name string

		Discounts Discounts

		ExpectedError bool
	}{
		{
			Name: "Valid",
			Discounts: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(50),
				},
			},
			ExpectedError: false,
		},
		{
			Name: "Invalid - more than 100% percentage discount",
			Discounts: Discounts{
				Percentage: &PercentageDiscount{
					Percentage: models.NewPercentage(110),
				},
			},
			ExpectedError: true,
		},
		{
			Name: "Valid - usage",
			Discounts: Discounts{
				Usage: &UsageDiscount{
					Quantity: decimal.NewFromInt(100),
				},
			},
			ExpectedError: false,
		},
		{
			Name: "Invalid - usage - negative",
			Discounts: Discounts{
				Usage: &UsageDiscount{
					Quantity: decimal.NewFromInt(-100),
				},
			},
			ExpectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := test.Discounts.ValidateForPrice(NewPriceFrom(
				UnitPrice{
					Amount: decimal.NewFromInt(100),
				},
			))
			if test.ExpectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
