package plan

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  map[string]string
		Right map[string]string

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Right: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Right: map[string]string{
				"key4": "value4",
				"key5": "value5",
				"key6": "value6",
			},
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := MetadataEqual(test.Left, test.Right)

			assert.Equal(t, test.ExpectedResult, result)
		})
	}
}

func TestDiscountsEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  []Discount
		Right []Discount

		ExpectedError  bool
		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			ExpectedError:  false,
			ExpectedResult: true,
		},
		{
			Name: "Left diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
			},
			ExpectedError:  false,
			ExpectedResult: false,
		},
		{
			Name: "Right diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			ExpectedError:  false,
			ExpectedResult: false,
		},
		{
			Name: "Complete diff",
			Left: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard1",
						"ratecard2",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(100),
					RateCards: []string{
						"ratecard3",
						"ratecard4",
					},
				}),
			},
			Right: []Discount{
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard5",
						"ratecard6",
					},
				}),
				NewDiscountFrom(PercentageDiscount{
					Percentage: decimal.NewFromInt(200),
					RateCards: []string{
						"ratecard7",
						"ratecard8",
					},
				}),
			},
			ExpectedError:  false,
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := DiscountsEqual(test.Left, test.Right)
			if test.ExpectedError {
				require.Errorf(t, err, "expected to fail")
			}
			require.NoErrorf(t, err, "expected to succeed")

			assert.Equal(t, test.ExpectedResult, result)
		})
	}
}
