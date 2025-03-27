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
		Discount      Discount
		ExpectedError bool
	}{
		{
			Name: "Valid - percentage",
			Discount: NewDiscountFrom(PercentageDiscount{
				Percentage: models.NewPercentage(99.9),
			}),
		},
		{
			Name: "Valid - usage",
			Discount: NewDiscountFrom(UsageDiscount{
				Usage: decimal.NewFromInt(100),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.Discount)
			require.NoError(t, err)

			t.Logf("Serialized Discount: %s", string(b))

			d := Discount{}
			err = json.Unmarshal(b, &d)
			require.NoError(t, err)

			assert.Equal(t, test.Discount, d)
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
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
				}),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
				}),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
				}),
				NewDiscountFrom(UsageDiscount{
					Usage: decimal.NewFromInt(100),
				}),
			},
			Right: []Discount{
				NewDiscountFrom(UsageDiscount{
					Usage: decimal.NewFromInt(200),
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
				}),
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
