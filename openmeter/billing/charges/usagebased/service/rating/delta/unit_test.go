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

func TestUnitDeltaInitialUsage(t *testing.T) {
	// Given:
	// - a unit price and initial metered usage
	// When:
	// - delta rating rates the first snapshot
	// Then:
	// - the full usage is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 50,
							Total:  50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 50,
					Total:  50,
				},
			},
		},
	})
}

func TestUnitDeltaAdditionalUsage(t *testing.T) {
	// Given:
	// - a unit price and prior usage already booked
	// When:
	// - delta rating rates a larger cumulative snapshot
	// Then:
	// - only the additional usage is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 50,
							Total:  50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 50,
					Total:  50,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 8,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 30,
					Total:  30,
				},
			},
		},
	})
}

func TestUnitDeltaNoAdditionalUsage(t *testing.T) {
	// Given:
	// - a unit price and prior usage already booked
	// When:
	// - delta rating rates an unchanged cumulative snapshot
	// Then:
	// - no additional detailed lines are produced
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 50,
							Total:  50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 50,
					Total:  50,
				},
			},
			{
				period:                periods.period2,
				meteredQuantity:       5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestUnitDeltaUsageDecrease(t *testing.T) {
	// Given:
	// - a unit price and prior usage already booked
	// When:
	// - delta rating rates a lower cumulative snapshot
	// Then:
	// - the usage decrease is booked as a matched negative delta
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 8,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               8,
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
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -30,
							Total:  -30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -30,
					Total:  -30,
				},
			},
		},
	})
}

func TestUnitDeltaUsageDropsToZero(t *testing.T) {
	// Given:
	// - a unit price and prior usage already booked
	// When:
	// - delta rating rates a current snapshot with no usage
	// Then:
	// - the previous usage line is reversed with a correction child reference
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 50,
							Total:  50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 50,
					Total:  50,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 0,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "unit-price-usage#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -50,
							Total:  -50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -50,
					Total:  -50,
				},
			},
		},
	})
}

func TestUnitDeltaRoundingCorrection(t *testing.T) {
	// Given:
	// - a unit price that rounds cumulative amounts to currency precision
	// - prior rounded usage is already booked
	// When:
	// - delta rating rates a larger cumulative snapshot
	// Then:
	// - the matched delta corrects the rounding difference between cumulative totals
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromFloat(0.001),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 333,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          0.001,
						Quantity:               333,
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
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          0.001,
						Quantity:               333,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 0.34,
							Total:  0.34,
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

func TestUnitDeltaRoundingNoVisibleDeltaProducesNoLines(t *testing.T) {
	// Given:
	// - a unit price whose cumulative totals round to zero
	// When:
	// - delta rating rates snapshots that remain invisible at currency precision
	// Then:
	// - zero-total detailed lines are dropped
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromFloat(0.001),
		}),
		phases: []deltaRatingPhase{
			{
				period:                periods.period1,
				meteredQuantity:       1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
			{
				period:                periods.period2,
				meteredQuantity:       2,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}
