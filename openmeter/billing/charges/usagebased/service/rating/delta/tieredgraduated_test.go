package delta

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestGraduatedTieredDeltaSameTierSubtractsNormally(t *testing.T) {
	// Given:
	// - a graduated-tiered price and prior usage in the same tier
	// When:
	// - delta rating rates a larger current snapshot in that tier
	// Then:
	// - only the additional tier quantity is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 3,
							Total:  3,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 3,
					Total:  3,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 2,
							Total:  2,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 2,
					Total:  2,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaCrossingTierBoundaryBooksOnlyNewTierUsage(t *testing.T) {
	// Given:
	// - a graduated-tiered price and prior usage in the first tier
	// When:
	// - the current cumulative snapshot crosses into the second tier
	// Then:
	// - only the remaining first-tier quantity and new second-tier quantity are booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 3,
							Total:  3,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 3,
					Total:  3,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 7,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 2,
							Total:  2,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          2,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 4,
							Total:  4,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 6,
					Total:  6,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaMultipleTierJumpBooksOnlyNewTierUsage(t *testing.T) {
	// Given:
	// - a graduated-tiered price and prior usage in the first tier
	// When:
	// - the current cumulative snapshot jumps across multiple tiers
	// Then:
	// - only the newly required quantities for each tier are booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 3,
							Total:  3,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 3,
					Total:  3,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 12,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 2,
							Total:  2,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          2,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          3,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 6,
							Total:  6,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 18,
					Total:  18,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaUsageDecreaseWithinSameTierReversesTierQuantity(t *testing.T) {
	// Given:
	// - a graduated-tiered price and prior usage in the first tier
	// When:
	// - the current cumulative snapshot decreases but stays in the same tier
	// Then:
	// - the tier line is emitted as a matched negative delta
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 5,
							Total:  5,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 5,
					Total:  5,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               -2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -2,
							Total:  -2,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -2,
					Total:  -2,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaUsageDecreaseAcrossTierBoundaryReversesRemovedTier(t *testing.T) {
	// Given:
	// - a graduated-tiered price and prior usage that reached the second tier
	// When:
	// - the current cumulative snapshot drops below the second-tier boundary
	// Then:
	// - the first tier is reduced and the removed second-tier line is reversed
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 7,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               5,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 5,
							Total:  5,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          2,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 4,
							Total:  4,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 9,
					Total:  9,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               -2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -2,
							Total:  -2,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage#correction:detailed_line_id=phase-1-line-2",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          2,
						Quantity:               -2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -4,
							Total:  -4,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -6,
					Total:  -6,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaFlatTierAlreadyBookedDoesNotRepeat(t *testing.T) {
	// Given:
	// - a graduated-tiered price with a flat tier already booked
	// When:
	// - the current cumulative snapshot remains inside the same flat tier
	// Then:
	// - only the usage delta is booked and the flat component is not repeated
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					FlatPrice: &productcatalog.PriceTierFlatPrice{
						Amount: alpacadecimal.NewFromInt(100),
					},
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 3,
							Total:  3,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          100,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 100,
							Total:  100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 103,
					Total:  103,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 4,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 1,
							Total:  1,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 1,
					Total:  1,
				},
			},
		},
	})
}

func TestGraduatedTieredDeltaCrossingFlatTierBoundaryBooksOnlyNewFlatComponent(t *testing.T) {
	// Given:
	// - a graduated-tiered price with a first-tier flat component already booked
	// When:
	// - the current cumulative snapshot crosses into a second flat tier
	// Then:
	// - the first-tier flat component is not repeated and the second-tier flat component is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
					FlatPrice: &productcatalog.PriceTierFlatPrice{
						Amount: alpacadecimal.NewFromInt(100),
					},
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					},
				},
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
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
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 3,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 3,
							Total:  3,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          100,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 100,
							Total:  100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 103,
					Total:  103,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 7,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          1,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 2,
							Total:  2,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          2,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 4,
							Total:  4,
						},
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          200,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 200,
							Total:  200,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 206,
					Total:  206,
				},
			},
		},
	})
}
