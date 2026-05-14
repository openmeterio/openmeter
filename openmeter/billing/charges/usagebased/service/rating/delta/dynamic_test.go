package delta

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestDynamicDeltaNoUsageProducesNoLines(t *testing.T) {
	// Given:
	// - a dynamic price and no metered usage
	// When:
	// - delta rating rates the current snapshot
	// Then:
	// - no detailed lines are produced
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
		}),
		phases: []deltaRatingPhase{
			{
				period:                periods.period1,
				meteredQuantity:       0,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestDynamicDeltaInitialUsageCreatesCurrentPeriodUsageLine(t *testing.T) {
	// Given:
	// - a dynamic price and initial metered usage
	// When:
	// - delta rating rates the first snapshot
	// Then:
	// - the full cumulative amount is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromFloat(1.33333),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          13.33,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 13.33,
							Total:  13.33,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 13.33,
					Total:  13.33,
				},
			},
		},
	})
}

func TestDynamicDeltaAdditionalUsageRepricesCumulativeAmount(t *testing.T) {
	// Given:
	// - a dynamic price whose per-unit amount is the cumulative metered amount
	// - a prior run already booked the earlier cumulative amount
	// When:
	// - delta rating rates a larger current snapshot
	// Then:
	// - the prior amount is reversed and the new cumulative amount is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 10,
					Total:  10,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
					{
						ChildUniqueReferenceID: "usage#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -10,
							Total:  -10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 5,
					Total:  5,
				},
			},
		},
	})
}

func TestDynamicDeltaNoAdditionalUsageAfterPriorBookingProducesNoLines(t *testing.T) {
	// Given:
	// - a dynamic price and a prior run already booked the same cumulative amount
	// When:
	// - delta rating rates an unchanged current snapshot
	// Then:
	// - no additional detailed lines are produced
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 10,
					Total:  10,
				},
			},
			{
				period:                periods.period2,
				meteredQuantity:       10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestDynamicDeltaRoundingCorrection(t *testing.T) {
	// Given:
	// - a dynamic price that rounds cumulative amounts to currency precision
	// - a prior run already booked the rounded earlier amount
	// When:
	// - delta rating rates a later cumulative snapshot
	// Then:
	// - the output corrects the rounding delta between cumulative snapshots
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromFloat(0.001),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 333,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          0.33,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 0.33,
							Total:  0.33,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 0.33,
					Total:  0.33,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 666,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          0.67,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 0.67,
							Total:  0.67,
						},
					},
					{
						ChildUniqueReferenceID: "usage#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          0.33,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -0.33,
							Total:  -0.33,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 0.34,
					Total:  0.34,
				},
			},
		},
	})
}

func TestDynamicDeltaMaximumSpendClampsCumulativeAmount(t *testing.T) {
	// Given:
	// - a dynamic price with maximum spend
	// - a prior run already booked usage below the maximum
	// When:
	// - delta rating rates a cumulative snapshot above the maximum
	// Then:
	// - the clamped current line and prior reversal produce only the remaining billable amount
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(10),
			Commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 8,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          80,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 80,
					Total:  80,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 12,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          120,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         120,
							DiscountsTotal: 20,
							Total:          100,
						},
					},
					{
						ChildUniqueReferenceID: "usage#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          80,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -80,
							Total:  -80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         40,
					DiscountsTotal: 20,
					Total:          20,
				},
			},
		},
	})
}

func TestDynamicDeltaMinimumCommitmentOnlyOnFinalPhase(t *testing.T) {
	// Given:
	// - a dynamic price with minimum commitment
	// - the first run is partial and the second run reaches the service-period end
	// When:
	// - delta rating rates both snapshots
	// Then:
	// - minimum commitment is ignored for the partial run and booked only on the final run
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
			Commitments: productcatalog.Commitments{
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 10,
					Total:  10,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.MinSpendChildUniqueReferenceID,
						Category:               stddetailedline.CategoryCommitment,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          90,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							ChargesTotal: 90,
							Total:        90,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					ChargesTotal: 90,
					Total:        90,
				},
			},
		},
	})
}
