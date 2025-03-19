package productcatalog

import (
	"testing"

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
			Name: "Valid",
			Discount: NewDiscountFrom(PercentageDiscount{
				Percentage: models.NewPercentage(99.9),
				RateCards: []string{
					"ratecard-1",
					"ratecard-2",
				},
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
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(100),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard5",
						"ratecard6",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: models.NewPercentage(200),
					RateCards: []string{
						"ratecard7",
						"ratecard8",
					},
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
