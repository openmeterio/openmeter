package rating

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

func TestUnitPriceSubstract(t *testing.T) {
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
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
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
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
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

func TestMinimumCommitmentSubstract(t *testing.T) {
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

func TestGraduatedTieredSubstract(t *testing.T) {
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
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
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

func TestSubtractRatedRunDetailsInvalidChildUniqueReferenceID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   string
	}{
		{
			name: "missing at separator",
			id:   "unit-price-usage",
		},
		{
			name: "invalid suffix shape",
			id:   "unit-price-usage@[2025-01-01T00:00:00Z]",
		},
		{
			name: "invalid period start",
			id:   "unit-price-usage@[not-a-time..2025-02-01T00:00:00Z]",
		},
		{
			name: "invalid period end",
			id:   "unit-price-usage@[2025-01-01T00:00:00Z..not-a-time]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a rated run details value with an invalid child unique reference id
			// when:
			// - we subtract it from an empty rated run details value
			// then:
			// - subtraction fails during child unique reference parsing

			_, err := SubtractRatedRunDetails(
				usagebased.DetailedLines{
					{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Namespace: "ns",
							Name:      "feature",
						}),
						Category:               stddetailedline.CategoryRegular,
						ChildUniqueReferenceID: tc.id,
						PaymentTerm:            productcatalog.InArrearsPaymentTerm,
						ServicePeriod: timeutil.ClosedPeriod{
							From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
							To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
						},
						Currency:      currencyx.Code("USD"),
						PerUnitAmount: alpacadecimal.NewFromInt(10),
						Quantity:      alpacadecimal.NewFromInt(1),
					},
				},
				usagebased.DetailedLines{},
			)

			require.Error(t, err)
		})
	}
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

			out, err := SubtractRatedRunDetails(current, previousWithMinimumCommitmentIgnored)

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
		makeRatedDetailedLineForTest("unit-price-usage", servicePeriod, 10, 5, stddetailedline.CategoryRegular, expectedTotals{
			Amount: 50,
			Total:  50,
		}),
		makeRatedDetailedLineForTest("unit-price-max-spend", servicePeriod, 1, 1, stddetailedline.CategoryRegular, expectedTotals{
			DiscountsTotal: 10,
			Total:          -10,
		}),
	}

	b := usagebased.DetailedLines{
		makeRatedDetailedLineForTest("unit-price-usage", servicePeriod, 10, 3, stddetailedline.CategoryRegular, expectedTotals{
			Amount: 30,
			Total:  30,
		}),
		makeRatedDetailedLineForTest("unit-price-max-spend", servicePeriod, 1, 2, stddetailedline.CategoryRegular, expectedTotals{
			DiscountsTotal: 20,
			Total:          -20,
		}),
	}

	// given:
	// - one detailed line component that grows between runs
	// - one detailed line component that shrinks between runs
	// when:
	// - we subtract the booked run details from the current run details
	// then:
	// - the result keeps both the positive and the negative delta lines

	out, err := SubtractRatedRunDetails(a, b)
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

	res.DetailedLines = withServicePeriodInDetailedLineChildUniqueReferenceIDs(res.DetailedLines, in.ServicePeriod)

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
		}
	})
}

type expectedDetailedLine struct {
	ChildUniqueReferenceID string
	Category               stddetailedline.Category
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

func makeRatedDetailedLineForTest(
	referenceID string,
	servicePeriod timeutil.ClosedPeriod,
	perUnitAmount float64,
	quantity float64,
	category stddetailedline.Category,
	lineTotals expectedTotals,
) usagebased.DetailedLine {
	return usagebased.DetailedLine{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace: "ns",
			Name:      "feature",
		}),
		Category:               category,
		ChildUniqueReferenceID: formatDetailedLineChildUniqueReferenceID(referenceID, servicePeriod),
		PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		ServicePeriod:          servicePeriod,
		Currency:               currencyx.Code("USD"),
		PerUnitAmount:          alpacadecimal.NewFromFloat(perUnitAmount),
		Quantity:               alpacadecimal.NewFromFloat(quantity),
		Totals: totals.Totals{
			Amount:              alpacadecimal.NewFromFloat(lineTotals.Amount),
			ChargesTotal:        alpacadecimal.NewFromFloat(lineTotals.ChargesTotal),
			DiscountsTotal:      alpacadecimal.NewFromFloat(lineTotals.DiscountsTotal),
			TaxesInclusiveTotal: alpacadecimal.NewFromFloat(lineTotals.TaxesInclusiveTotal),
			TaxesExclusiveTotal: alpacadecimal.NewFromFloat(lineTotals.TaxesExclusiveTotal),
			TaxesTotal:          alpacadecimal.NewFromFloat(lineTotals.TaxesTotal),
			CreditsTotal:        alpacadecimal.NewFromFloat(lineTotals.CreditsTotal),
			Total:               alpacadecimal.NewFromFloat(lineTotals.Total),
		},
	}
}
