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

func TestPackageDeltaInitialPartialPackage(t *testing.T) {
	// Given:
	// - a package price and usage below the first package boundary
	// When:
	// - delta rating rates the first snapshot
	// Then:
	// - one package is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
		},
	})
}

func TestPackageDeltaUsageWithinAlreadyBilledPackageProducesNoLines(t *testing.T) {
	// Given:
	// - a package price and one package already booked
	// When:
	// - later cumulative snapshots stay inside the same package
	// Then:
	// - no additional detailed lines are produced
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
			{
				period:                periods.period2,
				meteredQuantity:       9,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
			{
				period:                periods.period3,
				meteredQuantity:       10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestPackageDeltaCrossingPackageBoundaryBillsOnlyNewPackage(t *testing.T) {
	// Given:
	// - a package price and one package already booked
	// When:
	// - the current cumulative snapshot crosses into the next package
	// Then:
	// - only the newly required package is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 9,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 11,
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
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
		},
	})
}

func TestPackageDeltaMultiplePackageJumpBillsOnlyNewPackages(t *testing.T) {
	// Given:
	// - a package price and one package already booked
	// When:
	// - the current cumulative snapshot jumps across multiple package boundaries
	// Then:
	// - only the newly required packages are booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 31,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          15,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 45,
							Total:  45,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 45,
					Total:  45,
				},
			},
		},
	})
}

func TestPackageDeltaUsageDecreaseWithinSamePackageProducesNoLines(t *testing.T) {
	// Given:
	// - a package price and one package already booked
	// When:
	// - the current cumulative snapshot decreases but remains in the same package
	// Then:
	// - no correction is emitted because the rounded package count is unchanged
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 9,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
			{
				period:                periods.period2,
				meteredQuantity:       1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestPackageDeltaUsageDecreaseAcrossPackageBoundaryReversesPackage(t *testing.T) {
	// Given:
	// - a package price and two packages already booked
	// When:
	// - the current cumulative snapshot decreases to one package
	// Then:
	// - one package is reversed on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 11,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               2,
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
			{
				period:          periods.period2,
				meteredQuantity: 9,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          15,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -15,
							Total:  -15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -15,
					Total:  -15,
				},
			},
		},
	})
}

func TestPackageDeltaUsageDropsToZeroReversesBookedPackage(t *testing.T) {
	// Given:
	// - a package price and one package already booked
	// When:
	// - the current cumulative snapshot drops to zero usage
	// Then:
	// - the previous package is reversed with a correction child reference
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			QuantityPerPackage: alpacadecimal.NewFromInt(10),
			Amount:             alpacadecimal.NewFromInt(15),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          15,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 15,
							Total:  15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 15,
					Total:  15,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 0,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "usage#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          15,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -15,
							Total:  -15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -15,
					Total:  -15,
				},
			},
		},
	})
}
