package subtract

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestUnitPriceSubtract(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	testCases := []subtractRatedRunDetailsTestCase{
		{
			name: "unit price returns cumulative delta",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
			previousMeteredQuantity: 3,
			newMeteredQuantity:      5,
			expectedTotals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          10,
					Quantity:               2,
					Totals: expectedTotals{
						Amount: 20,
						Total:  20,
					},
				},
			},
		},
		{
			name: "unit price subtraction corrects for per-run rounding",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(0.001),
			}),
			previousMeteredQuantity: 333,
			newMeteredQuantity:      666,
			expectedTotals: expectedTotals{
				Amount: 0.34,
				Total:  0.34,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          0.001,
					Quantity:               333,
					Totals: expectedTotals{
						Amount: 0.34,
						Total:  0.34,
					},
				},
			},
		},
		{
			name: "unit price with no new usage returns no detailed lines",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
			previousMeteredQuantity: 5,
			newMeteredQuantity:      5,
			expectedTotals:          expectedTotals{},
			expectedLines:           []expectedDetailedLine{},
		},
		{
			name: "unit price drops detailed line when rounding makes delta zero",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(0.001),
			}),
			previousMeteredQuantity: 100,
			newMeteredQuantity:      102,
			expectedTotals:          expectedTotals{},
			expectedLines:           []expectedDetailedLine{},
		},
		{
			name: "unit price with lower new usage returns negative delta (e.g. min meter)",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
			previousMeteredQuantity: 5,
			newMeteredQuantity:      0,
			expectedTotals: expectedTotals{
				Amount: -50,
				Total:  -50,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "unit-price-usage#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          10,
					Quantity:               -5,
					Totals: expectedTotals{
						Amount: -50,
						Total:  -50,
					},
				},
			},
		},
		{
			name: "unit price subtraction keeps maximum spend partial additional billable amount",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
				},
			}),
			previousMeteredQuantity: 5,
			newMeteredQuantity:      15,
			expectedTotals: expectedTotals{
				Amount:         100,
				DiscountsTotal: 50,
				Total:          50,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          10,
					Quantity:               10,
					Totals: expectedTotals{
						Amount:         100,
						DiscountsTotal: 50,
						Total:          50,
					},
				},
			},
		},
		{
			name: "unit price subtraction keeps capped zero-total delta when maximum spend is already hit on both sides",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
				},
			}),
			previousMeteredQuantity: 15,
			newMeteredQuantity:      20,
			expectedTotals: expectedTotals{
				Amount:         50,
				DiscountsTotal: 50,
				Total:          0,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          10,
					Quantity:               5,
					Totals: expectedTotals{
						Amount:         50,
						DiscountsTotal: 50,
						Total:          0,
					},
				},
			},
		},
		{
			name: "unit price returns small visible negative rounded delta",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(0.001),
			}),
			previousMeteredQuantity: 6,
			newMeteredQuantity:      0,
			expectedTotals: expectedTotals{
				Amount: -0.01,
				Total:  -0.01,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "unit-price-usage#reversal:category=regular:payment_term=in_arrears:per_unit_amount=0.001",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          0.001,
					Quantity:               -6,
					Totals: expectedTotals{
						Amount: -0.01,
						Total:  -0.01,
					},
				},
			},
		},
	}

	runSubtractRatedRunDetailsTestCases(t, servicePeriod, testCases)
}

func TestMinimumCommitmentSubtract(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	testCases := []subtractRatedRunDetailsTestCase{
		{
			name: "minimum commitment appears when only current has it",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
				},
			}),
			previousMeteredQuantity: 0,
			newMeteredQuantity:      0,
			expectedTotals: expectedTotals{
				ChargesTotal: 100,
				Total:        100,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.MinSpendChildUniqueReferenceID,
					Category:               stddetailedline.CategoryCommitment,
					PerUnitAmount:          100,
					Quantity:               1,
					Totals: expectedTotals{
						ChargesTotal: 100,
						Total:        100,
					},
				},
			},
		},
		{
			name: "usage and commitment can both appear in one subtraction result",
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
				},
			}),
			previousMeteredQuantity: 0,
			newMeteredQuantity:      5,
			expectedTotals: expectedTotals{
				Amount:       50,
				ChargesTotal: 50,
				Total:        100,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: billingrating.MinSpendChildUniqueReferenceID,
					Category:               stddetailedline.CategoryCommitment,
					PerUnitAmount:          50,
					Quantity:               1,
					Totals: expectedTotals{
						ChargesTotal: 50,
						Total:        50,
					},
				},
				{
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          10,
					Quantity:               5,
					Totals: expectedTotals{
						Amount: 50,
						Total:  50,
					},
				},
			},
		},
	}

	runSubtractRatedRunDetailsTestCases(t, servicePeriod, testCases)
}

func TestGraduatedTieredSubtract(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	graduatedUnitPrice := *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(20)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(3),
				},
			},
		},
	})

	graduatedFlatAndUnitPrice := *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromInt(100),
				},
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(20)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromInt(200),
				},
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(3),
				},
			},
		},
	})

	testCases := []subtractRatedRunDetailsTestCase{
		{
			name:                    "graduated tiered same tier returns tier delta",
			price:                   graduatedUnitPrice,
			previousMeteredQuantity: 3,
			newMeteredQuantity:      7,
			expectedTotals: expectedTotals{
				Amount: 4,
				Total:  4,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          1,
					Quantity:               4,
					Totals: expectedTotals{
						Amount: 4,
						Total:  4,
					},
				},
			},
		},
		{
			name:                    "graduated tiered crossing second tier keeps per tier deltas",
			price:                   graduatedUnitPrice,
			previousMeteredQuantity: 8,
			newMeteredQuantity:      15,
			expectedTotals: expectedTotals{
				Amount: 12,
				Total:  12,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          1,
					Quantity:               2,
					Totals: expectedTotals{
						Amount: 2,
						Total:  2,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          2,
					Quantity:               5,
					Totals: expectedTotals{
						Amount: 10,
						Total:  10,
					},
				},
			},
		},
		{
			name:                    "graduated tiered crossing multiple tiers keeps all tier deltas",
			price:                   graduatedUnitPrice,
			previousMeteredQuantity: 5,
			newMeteredQuantity:      25,
			expectedTotals: expectedTotals{
				Amount: 40,
				Total:  40,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          1,
					Quantity:               5,
					Totals: expectedTotals{
						Amount: 5,
						Total:  5,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          2,
					Quantity:               10,
					Totals: expectedTotals{
						Amount: 20,
						Total:  20,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          3,
					Quantity:               5,
					Totals: expectedTotals{
						Amount: 15,
						Total:  15,
					},
				},
			},
		},
		{
			name:                    "graduated tiered decrease across tier boundary returns negative tier deltas",
			price:                   graduatedUnitPrice,
			previousMeteredQuantity: 15,
			newMeteredQuantity:      8,
			expectedTotals: expectedTotals{
				Amount: -12,
				Total:  -12,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          1,
					Quantity:               -2,
					Totals: expectedTotals{
						Amount: -2,
						Total:  -2,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage#reversal:category=regular:payment_term=in_arrears:per_unit_amount=2",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          2,
					Quantity:               -5,
					Totals: expectedTotals{
						Amount: -10,
						Total:  -10,
					},
				},
			},
		},
		{
			name:                    "graduated tiered flat and unit components keep cumulative deltas separately",
			price:                   graduatedFlatAndUnitPrice,
			previousMeteredQuantity: 0,
			newMeteredQuantity:      15,
			expectedTotals: expectedTotals{
				Amount: 220,
				Total:  220,
			},
			expectedLines: []expectedDetailedLine{
				{
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          1,
					Quantity:               10,
					Totals: expectedTotals{
						Amount: 10,
						Total:  10,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          200,
					Quantity:               1,
					Totals: expectedTotals{
						Amount: 200,
						Total:  200,
					},
				},
				{
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
					Category:               stddetailedline.CategoryRegular,
					PerUnitAmount:          2,
					Quantity:               5,
					Totals: expectedTotals{
						Amount: 10,
						Total:  10,
					},
				},
			},
		},
	}

	runSubtractRatedRunDetailsTestCases(t, servicePeriod, testCases)
}

func TestSubtractRatedRunDetailsMatchesByPricerReferenceIDAndPreservesChildUniqueReferenceID(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:            "unit-price-usage",
		childUniqueReferenceID: "current-child-reference",
		servicePeriod:          servicePeriod,
		perUnitAmount:          10,
		quantity:               5,
		totals: expectedTotals{
			Amount: 50,
			Total:  50,
		},
	})

	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:            "unit-price-usage",
		childUniqueReferenceID: "previous-child-reference",
		servicePeriod:          servicePeriod,
		perUnitAmount:          10,
		quantity:               3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})

	// given:
	// - a current and previous detailed line with different child references
	// - both lines share the same pricer reference
	// when:
	// - we subtract the previous detailed line from the current detailed line
	// then:
	// - subtraction matches by pricer reference
	// - the output keeps the current child reference as persistence identity

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "current-child-reference",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
	}, toExpectedDetailedLines(out))
}

func TestSubtractRatedRunDetailsDoesNotMatchDifferentPricerReferenceIDs(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:            "current-pricer-reference",
		childUniqueReferenceID: "shared-child-reference",
		servicePeriod:          servicePeriod,
		perUnitAmount:          10,
		quantity:               5,
		totals: expectedTotals{
			Amount: 50,
			Total:  50,
		},
	})

	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:            "previous-pricer-reference",
		childUniqueReferenceID: "shared-child-reference",
		servicePeriod:          servicePeriod,
		perUnitAmount:          10,
		quantity:               3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})

	// given:
	// - a current and previous detailed line with the same child reference
	// - both lines have different pricer references
	// when:
	// - we subtract the previous detailed line from the current detailed line
	// then:
	// - subtraction does not match them arithmetically
	// - the previous-only reversal gets a distinct child reference

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "shared-child-reference",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               5,
			Totals: expectedTotals{
				Amount: 50,
				Total:  50,
			},
		},
		{
			ChildUniqueReferenceID: "shared-child-reference#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               -3,
			Totals: expectedTotals{
				Amount: -30,
				Total:  -30,
			},
		},
	}, toExpectedDetailedLines(out))
}

func TestSubtractRatedRunDetailsRejectsCurrencyMismatch(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "unit-price-usage",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      5,
		totals: expectedTotals{
			Amount: 50,
			Total:  50,
		},
	})
	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "unit-price-usage",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})
	previous.Currency = currencyx.Code("EUR")

	// given:
	// - current and previous detailed lines with different currencies
	// when:
	// - we subtract the previous detailed lines from the current detailed lines
	// then:
	// - subtraction rejects the invocation instead of treating currency as an arithmetic key

	_, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.ErrorContains(t, err, "current and previous detailed lines: currency mismatch: USD != EUR")
}

func TestSubtractRatedRunDetailsPreservesMatchedCurrentServicePeriod(t *testing.T) {
	t.Parallel()

	firstPeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	secondPeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "unit-price-usage",
		servicePeriod: firstPeriod,
		perUnitAmount: 10,
		quantity:      5,
		totals: expectedTotals{
			Amount: 50,
			Total:  50,
		},
	})
	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "unit-price-usage",
		servicePeriod: secondPeriod,
		perUnitAmount: 10,
		quantity:      3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})

	// given:
	// - two detailed lines with the same pricer reference id but different service periods
	// when:
	// - we subtract them
	// then:
	// - the arithmetic preserves the current line service period for the matched output line

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, firstPeriod, out[0].ServicePeriod)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "unit-price-usage",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
	}, toExpectedDetailedLines(out))
}

func TestSubtractRatedRunDetailsPreservesIndexes(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	currentOnly := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "a-current-only",
		servicePeriod: servicePeriod,
		index:         lo.ToPtr(11),
		perUnitAmount: 10,
		quantity:      2,
		totals: expectedTotals{
			Amount: 20,
			Total:  20,
		},
	})

	currentMatched := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "b-matched",
		servicePeriod: servicePeriod,
		index:         lo.ToPtr(22),
		perUnitAmount: 10,
		quantity:      5,
		totals: expectedTotals{
			Amount: 50,
			Total:  50,
		},
	})

	previousMatched := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "b-matched",
		servicePeriod: servicePeriod,
		index:         lo.ToPtr(222),
		perUnitAmount: 10,
		quantity:      3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})

	previousOnly := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "c-previous-only",
		servicePeriod: servicePeriod,
		index:         lo.ToPtr(33),
		perUnitAmount: 10,
		quantity:      4,
		totals: expectedTotals{
			Amount: 40,
			Total:  40,
		},
	})

	// given:
	// - a current-only line with an index
	// - a matched line where current and previous have different indexes
	// - a previous-only line with an index
	// when:
	// - we subtract previous detailed lines from current detailed lines
	// then:
	// - current-only output keeps the current index
	// - matched output keeps the current index
	// - previous-only reversal keeps the previous index
	// - subtraction does not invent new indexes

	out, err := SubtractRatedRunDetails(
		usagebased.DetailedLines{currentOnly, currentMatched},
		usagebased.DetailedLines{previousMatched, previousOnly},
		NewMockUniqueReferenceIDGenerator(t),
	)
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "a-current-only",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
		{
			ChildUniqueReferenceID: "b-matched",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
		{
			ChildUniqueReferenceID: "c-previous-only#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               -4,
			Totals: expectedTotals{
				Amount: -40,
				Total:  -40,
			},
		},
	}, toExpectedDetailedLines(out))

	require.Len(t, out, 3)
	require.Equal(t, lo.ToPtr(11), out[0].Index)
	require.Equal(t, lo.ToPtr(22), out[1].Index)
	require.Equal(t, lo.ToPtr(33), out[2].Index)
}

func TestSubtractRatedRunDetailsKeepsRepricingAsSeparateLines(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 5,
		quantity:      16,
		totals: expectedTotals{
			Amount: 80,
			Total:  80,
		},
	})
	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      15,
		totals: expectedTotals{
			Amount: 150,
			Total:  150,
		},
	})

	// given:
	// - a volume-tiered line that keeps the same child reference id but changes per-unit amount
	// when:
	// - we subtract the previous rating from the current rating
	// then:
	// - the result keeps a full positive current line and a full negative previous line instead of a quantity-only delta

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "volume-tiered-price",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          5,
			Quantity:               16,
			Totals: expectedTotals{
				Amount: 80,
				Total:  80,
			},
		},
		{
			ChildUniqueReferenceID: "volume-tiered-price#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               -15,
			Totals: expectedTotals{
				Amount: -150,
				Total:  -150,
			},
		},
	}, toExpectedDetailedLines(out))
}

func TestSubtractRatedRunDetailsDisambiguatesPreviousOnlyRepricingReversalChildUniqueReferenceID(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 5,
		quantity:      16,
		totals: expectedTotals{
			Amount: 80,
			Total:  80,
		},
	})
	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      15,
		totals: expectedTotals{
			Amount: 150,
			Total:  150,
		},
	})

	// given:
	// - repricing creates a positive current line and a negative previous line
	// - both input lines have the same child reference
	// when:
	// - we subtract the previous detailed lines from the current detailed lines
	// then:
	// - subtraction keeps the current child reference
	// - subtraction derives a stable child reference for the previous-only reversal

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Len(t, out, 2)
	require.Equal(t, "volume-tiered-price", out[0].ChildUniqueReferenceID)
	require.Equal(t, "volume-tiered-price#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10", out[1].ChildUniqueReferenceID)
}

func TestSubtractRatedRunDetailsAllowsDuplicateChildUniqueReferenceIDsWhenValidationIgnored(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 5,
		quantity:      16,
		totals: expectedTotals{
			Amount: 80,
			Total:  80,
		},
	})
	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "volume-tiered-price",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      15,
		totals: expectedTotals{
			Amount: 150,
			Total:  150,
		},
	})

	_, err := SubtractRatedRunDetails(
		usagebased.DetailedLines{current},
		usagebased.DetailedLines{previous},
		passthroughUniqueReferenceIDGenerator{},
	)
	require.ErrorContains(t, err, "duplicate child unique reference id: volume-tiered-price")

	out, err := SubtractRatedRunDetails(
		usagebased.DetailedLines{current},
		usagebased.DetailedLines{previous},
		passthroughUniqueReferenceIDGenerator{},
		WithUniqueReferenceIDValidationIgnored(),
	)
	require.NoError(t, err)
	require.Len(t, out, 2)
	require.Equal(t, "volume-tiered-price", out[0].ChildUniqueReferenceID)
	require.Equal(t, "volume-tiered-price", out[1].ChildUniqueReferenceID)
}

func TestVolumeTieredSubtract(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("same tier subtracts normally", func(t *testing.T) {
		current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 10,
			quantity:      12,
			totals: expectedTotals{
				Amount: 120,
				Total:  120,
			},
		})
		previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 10,
			quantity:      10,
			totals: expectedTotals{
				Amount: 100,
				Total:  100,
			},
		})

		// given:
		// - a volume-tiered line that remains in the same tier and keeps the same per-unit amount
		// when:
		// - we subtract the previous rating from the current rating
		// then:
		// - the result is a normal quantity and totals delta

		out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
		require.NoError(t, err)
		require.Equal(t, []expectedDetailedLine{
			{
				ChildUniqueReferenceID: "volume-tiered-price",
				Category:               stddetailedline.CategoryRegular,
				PerUnitAmount:          10,
				Quantity:               2,
				Totals: expectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
		}, toExpectedDetailedLines(out))
	})

	t.Run("same ref and per unit current lines aggregate before subtract", func(t *testing.T) {
		currentA := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 5,
			quantity:      10,
			totals: expectedTotals{
				Amount: 50,
				Total:  50,
			},
		})
		currentB := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 5,
			quantity:      6,
			totals: expectedTotals{
				Amount: 30,
				Total:  30,
			},
		})
		previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 5,
			quantity:      12,
			totals: expectedTotals{
				Amount: 60,
				Total:  60,
			},
		})

		// given:
		// - two current volume-tiered lines with the same pricer reference and per-unit amount
		// - one previous line with the same pricer reference and per-unit amount
		// when:
		// - we subtract the previous rating from the current rating
		// then:
		// - current lines are aggregated before subtraction

		out, err := SubtractRatedRunDetails(
			usagebased.DetailedLines{currentA, currentB},
			usagebased.DetailedLines{previous},
			NewMockUniqueReferenceIDGenerator(t),
		)
		require.NoError(t, err)
		require.Equal(t, []expectedDetailedLine{
			{
				ChildUniqueReferenceID: "volume-tiered-price",
				Category:               stddetailedline.CategoryRegular,
				PerUnitAmount:          5,
				Quantity:               4,
				Totals: expectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
		}, toExpectedDetailedLines(out))
	})

	t.Run("volume tiered downgrade repricing keeps separate lines", func(t *testing.T) {
		current := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 10,
			quantity:      15,
			totals: expectedTotals{
				Amount: 150,
				Total:  150,
			},
		})
		previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "volume-tiered-price",
			servicePeriod: servicePeriod,
			perUnitAmount: 5,
			quantity:      16,
			totals: expectedTotals{
				Amount: 80,
				Total:  80,
			},
		})

		// given:
		// - a volume-tiered line that moves back to a higher per-unit amount
		// when:
		// - we subtract the previous rating from the current rating
		// then:
		// - the result keeps a full positive current line and a full negative previous line

		out, err := SubtractRatedRunDetails(usagebased.DetailedLines{current}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
		require.NoError(t, err)
		require.Equal(t, []expectedDetailedLine{
			{
				ChildUniqueReferenceID: "volume-tiered-price#reversal:category=regular:payment_term=in_arrears:per_unit_amount=5",
				Category:               stddetailedline.CategoryRegular,
				PerUnitAmount:          5,
				Quantity:               -16,
				Totals: expectedTotals{
					Amount: -80,
					Total:  -80,
				},
			},
			{
				ChildUniqueReferenceID: "volume-tiered-price",
				Category:               stddetailedline.CategoryRegular,
				PerUnitAmount:          10,
				Quantity:               15,
				Totals: expectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
		}, toExpectedDetailedLines(out))
	})
}

func TestSubtractRatedRunDetailsAcceptsNegativeCorrectionInputs(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	previous := makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
		referenceID:   "unit-price-usage",
		servicePeriod: servicePeriod,
		perUnitAmount: 10,
		quantity:      3,
		totals: expectedTotals{
			Amount: 30,
			Total:  30,
		},
	})
	previous.Quantity = previous.Quantity.Neg()
	previous.Totals = previous.Totals.Neg()

	// given:
	// - an already billed correction line with a negative quantity
	// when:
	// - we subtract it from an empty current rating
	// then:
	// - subtraction treats it as arithmetic input and reverses it

	out, err := SubtractRatedRunDetails(usagebased.DetailedLines{}, usagebased.DetailedLines{previous}, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "unit-price-usage#reversal:category=regular:payment_term=in_arrears:per_unit_amount=10",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               3,
			Totals: expectedTotals{
				Amount: 30,
				Total:  30,
			},
		},
	}, toExpectedDetailedLines(out))
}

type rateRunDetailsForTestInput struct {
	ServicePeriod   timeutil.ClosedPeriod
	Price           productcatalog.Price
	MeteredQuantity float64
}

type subtractRatedRunDetailsTestCase struct {
	name string

	price                   productcatalog.Price
	previousMeteredQuantity float64
	newMeteredQuantity      float64

	expectedTotals expectedTotals
	expectedLines  []expectedDetailedLine
}

func runSubtractRatedRunDetailsTestCases(
	t *testing.T,
	servicePeriod timeutil.ClosedPeriod,
	testCases []subtractRatedRunDetailsTestCase,
) {
	t.Helper()

	ratingService := billingratingservice.New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a fixed service period
			// - a usage-based price for the testcase
			// - a previously booked cumulative metered quantity, rated without minimum commitment
			// - a newer cumulative metered quantity for the same period, rated with default mutators

			previousWithMinimumCommitmentIgnored := rateRunDetailsForTest(t, ratingService, rateRunDetailsForTestInput{
				ServicePeriod:   servicePeriod,
				Price:           tc.price,
				MeteredQuantity: tc.previousMeteredQuantity,
			}, billingrating.WithMinimumCommitmentIgnored())

			current := rateRunDetailsForTest(t, ratingService, rateRunDetailsForTestInput{
				ServicePeriod:   servicePeriod,
				Price:           tc.price,
				MeteredQuantity: tc.newMeteredQuantity,
			})

			// when:
			// - we subtract the previously booked rated run details from the new cumulative rated run details

			out, err := SubtractRatedRunDetails(current, previousWithMinimumCommitmentIgnored, NewMockUniqueReferenceIDGenerator(t))

			// then:
			// - the result contains only the delta totals and delta detailed lines

			require.NoError(t, err)
			require.Equal(t, tc.expectedTotals, toExpectedTotals(totalsFromExpectedDetailedLines(toExpectedDetailedLines(out))))
			require.Equal(t, tc.expectedLines, toExpectedDetailedLines(out))
		})
	}
}

func TestSubtractRatedRunDetailsMixedPositiveAndNegativeLines(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	a := usagebased.DetailedLines{
		makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "unit-price-usage",
			servicePeriod: servicePeriod,
			perUnitAmount: 10,
			quantity:      5,
			totals: expectedTotals{
				Amount: 50,
				Total:  50,
			},
		}),
		makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "unit-price-max-spend",
			servicePeriod: servicePeriod,
			perUnitAmount: 1,
			quantity:      1,
			totals: expectedTotals{
				DiscountsTotal: 10,
				Total:          -10,
			},
		}),
	}

	b := usagebased.DetailedLines{
		makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "unit-price-usage",
			servicePeriod: servicePeriod,
			perUnitAmount: 10,
			quantity:      3,
			totals: expectedTotals{
				Amount: 30,
				Total:  30,
			},
		}),
		makeRatedDetailedLineForTest(ratedDetailedLineForTestInput{
			referenceID:   "unit-price-max-spend",
			servicePeriod: servicePeriod,
			perUnitAmount: 1,
			quantity:      2,
			totals: expectedTotals{
				DiscountsTotal: 20,
				Total:          -20,
			},
		}),
	}

	// given:
	// - one detailed line component that grows between runs
	// - one detailed line component that shrinks between runs
	// when:
	// - we subtract the booked run details from the current run details
	// then:
	// - the result keeps both the positive and the negative delta lines

	out, err := SubtractRatedRunDetails(a, b, NewMockUniqueReferenceIDGenerator(t))
	require.NoError(t, err)
	require.Equal(t, []expectedDetailedLine{
		{
			ChildUniqueReferenceID: "unit-price-max-spend",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          1,
			Quantity:               -1,
			Totals: expectedTotals{
				DiscountsTotal: -10,
				Total:          10,
			},
		},
		{
			ChildUniqueReferenceID: "unit-price-usage",
			Category:               stddetailedline.CategoryRegular,
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: expectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
	}, toExpectedDetailedLines(out))
}

func rateRunDetailsForTest(t *testing.T, ratingService billingrating.Service, in rateRunDetailsForTestInput, opts ...billingrating.GenerateDetailedLinesOption) usagebased.DetailedLines {
	t.Helper()

	intent := usagebased.RateableIntent{
		Intent: usagebased.Intent{
			Intent: meta.Intent{
				Name:              "feature",
				Currency:          currencyx.Code("USD"),
				ServicePeriod:     in.ServicePeriod,
				FullServicePeriod: in.ServicePeriod,
				BillingPeriod:     in.ServicePeriod,
			},
			FeatureKey: "feature",
			Price:      in.Price,
		},
		ServicePeriod: in.ServicePeriod,
		MeterValue:    alpacadecimal.NewFromFloat(in.MeteredQuantity),
	}

	res, err := ratingService.GenerateDetailedLines(intent, opts...)
	require.NoError(t, err)

	return usageBasedDetailedLinesForTest(res.DetailedLines, in.ServicePeriod)
}

func usageBasedDetailedLinesForTest(lines billingrating.DetailedLines, servicePeriod timeutil.ClosedPeriod) usagebased.DetailedLines {
	return lo.Map(lines, func(line billingrating.DetailedLine, _ int) usagebased.DetailedLine {
		period := servicePeriod
		if line.Period != nil {
			period = *line.Period
		}

		category := line.Category
		if category == "" {
			category = stddetailedline.CategoryRegular
		}

		paymentTerm := lo.CoalesceOrEmpty(line.PaymentTerm, productcatalog.InArrearsPaymentTerm)

		return usagebased.DetailedLine{
			PricerReferenceID: line.ChildUniqueReferenceID,
			Base: stddetailedline.Base{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: "ns",
					Name:      line.Name,
				}),
				Category:               category,
				ChildUniqueReferenceID: line.ChildUniqueReferenceID,
				PaymentTerm:            paymentTerm,
				ServicePeriod:          period,
				Currency:               currencyx.Code("USD"),
				PerUnitAmount:          line.PerUnitAmount,
				Quantity:               line.Quantity,
				Totals:                 line.Totals,
			},
		}
	})
}

type expectedDetailedLine struct {
	ChildUniqueReferenceID string
	Category               stddetailedline.Category
	ServicePeriod          *timeutil.ClosedPeriod
	CorrectsRunID          *string
	PerUnitAmount          float64
	Quantity               float64
	Totals                 expectedTotals
}

type expectedTotals struct {
	Amount              float64
	ChargesTotal        float64
	DiscountsTotal      float64
	TaxesInclusiveTotal float64
	TaxesExclusiveTotal float64
	TaxesTotal          float64
	CreditsTotal        float64
	Total               float64
}

func toExpectedDetailedLines(lines usagebased.DetailedLines) []expectedDetailedLine {
	return lo.Map(lines, func(line usagebased.DetailedLine, _ int) expectedDetailedLine {
		return expectedDetailedLine{
			ChildUniqueReferenceID: line.ChildUniqueReferenceID,
			Category:               line.Category,
			CorrectsRunID:          line.CorrectsRunID,
			PerUnitAmount:          line.PerUnitAmount.InexactFloat64(),
			Quantity:               line.Quantity.InexactFloat64(),
			Totals:                 toExpectedTotals(line.Totals),
		}
	})
}

func toExpectedDetailedLinesWithServicePeriod(lines usagebased.DetailedLines) []expectedDetailedLine {
	return lo.Map(lines, func(line usagebased.DetailedLine, _ int) expectedDetailedLine {
		return expectedDetailedLine{
			ChildUniqueReferenceID: line.ChildUniqueReferenceID,
			Category:               line.Category,
			ServicePeriod:          lo.ToPtr(line.ServicePeriod),
			CorrectsRunID:          line.CorrectsRunID,
			PerUnitAmount:          line.PerUnitAmount.InexactFloat64(),
			Quantity:               line.Quantity.InexactFloat64(),
			Totals:                 toExpectedTotals(line.Totals),
		}
	})
}

func toExpectedTotals(in totals.Totals) expectedTotals {
	return expectedTotals{
		Amount:              in.Amount.InexactFloat64(),
		ChargesTotal:        in.ChargesTotal.InexactFloat64(),
		DiscountsTotal:      in.DiscountsTotal.InexactFloat64(),
		TaxesInclusiveTotal: in.TaxesInclusiveTotal.InexactFloat64(),
		TaxesExclusiveTotal: in.TaxesExclusiveTotal.InexactFloat64(),
		TaxesTotal:          in.TaxesTotal.InexactFloat64(),
		CreditsTotal:        in.CreditsTotal.InexactFloat64(),
		Total:               in.Total.InexactFloat64(),
	}
}

func totalsFromExpectedDetailedLines(lines []expectedDetailedLine) totals.Totals {
	out := totals.Totals{}

	for _, line := range lines {
		out = out.Add(totals.Totals{
			Amount:              alpacadecimal.NewFromFloat(line.Totals.Amount),
			ChargesTotal:        alpacadecimal.NewFromFloat(line.Totals.ChargesTotal),
			DiscountsTotal:      alpacadecimal.NewFromFloat(line.Totals.DiscountsTotal),
			TaxesInclusiveTotal: alpacadecimal.NewFromFloat(line.Totals.TaxesInclusiveTotal),
			TaxesExclusiveTotal: alpacadecimal.NewFromFloat(line.Totals.TaxesExclusiveTotal),
			TaxesTotal:          alpacadecimal.NewFromFloat(line.Totals.TaxesTotal),
			CreditsTotal:        alpacadecimal.NewFromFloat(line.Totals.CreditsTotal),
			Total:               alpacadecimal.NewFromFloat(line.Totals.Total),
		})
	}

	return out
}

type ratedDetailedLineForTestInput struct {
	referenceID            string
	childUniqueReferenceID string
	servicePeriod          timeutil.ClosedPeriod
	index                  *int
	perUnitAmount          float64
	quantity               float64
	category               stddetailedline.Category
	totals                 expectedTotals
}

type passthroughUniqueReferenceIDGenerator struct{}

func (passthroughUniqueReferenceIDGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func (passthroughUniqueReferenceIDGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
	return current.ChildUniqueReferenceID, nil
}

func (passthroughUniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
	return line.ChildUniqueReferenceID, nil
}

func makeRatedDetailedLineForTest(in ratedDetailedLineForTestInput) usagebased.DetailedLine {
	category := in.category
	if category == "" {
		category = stddetailedline.CategoryRegular
	}

	childUniqueReferenceID := in.childUniqueReferenceID
	if childUniqueReferenceID == "" {
		childUniqueReferenceID = in.referenceID
	}

	return usagebased.DetailedLine{
		PricerReferenceID: in.referenceID,
		Base: stddetailedline.Base{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				Name:      "feature",
			}),
			Category:               category,
			ChildUniqueReferenceID: childUniqueReferenceID,
			Index:                  in.index,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			ServicePeriod:          in.servicePeriod,
			Currency:               currencyx.Code("USD"),
			PerUnitAmount:          alpacadecimal.NewFromFloat(in.perUnitAmount),
			Quantity:               alpacadecimal.NewFromFloat(in.quantity),
			Totals: totals.Totals{
				Amount:              alpacadecimal.NewFromFloat(in.totals.Amount),
				ChargesTotal:        alpacadecimal.NewFromFloat(in.totals.ChargesTotal),
				DiscountsTotal:      alpacadecimal.NewFromFloat(in.totals.DiscountsTotal),
				TaxesInclusiveTotal: alpacadecimal.NewFromFloat(in.totals.TaxesInclusiveTotal),
				TaxesExclusiveTotal: alpacadecimal.NewFromFloat(in.totals.TaxesExclusiveTotal),
				TaxesTotal:          alpacadecimal.NewFromFloat(in.totals.TaxesTotal),
				CreditsTotal:        alpacadecimal.NewFromFloat(in.totals.CreditsTotal),
				Total:               alpacadecimal.NewFromFloat(in.totals.Total),
			},
		},
	}
}
