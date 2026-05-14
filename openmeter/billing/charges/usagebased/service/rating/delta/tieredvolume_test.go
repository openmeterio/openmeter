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

func TestVolumeTieredDeltaSameUnitTierSubtractsNormally(t *testing.T) {
	// Given:
	// - a volume-tiered unit price and prior usage in the same tier
	// When:
	// - delta rating rates a larger current snapshot in that tier
	// Then:
	// - only the additional quantity is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()
	price := volumeUnitTierPrice()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: price,
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 12,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               12,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 120,
							Total:  120,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 120,
					Total:  120,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 14,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 20,
							Total:  20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaUnitTierCrossingRepricesCurrentPeriod(t *testing.T) {
	// Given:
	// - a volume-tiered unit price and prior usage in the first tier
	// When:
	// - the current cumulative snapshot crosses into the next volume tier
	// Then:
	// - the first-tier line is reversed and the repriced cumulative tier line is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()
	price := volumeUnitTierPrice()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: price,
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 16,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
					{
						ChildUniqueReferenceID: "volume-tiered-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
			{
				period:                periods.period3,
				meteredQuantity:       16,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}

func TestVolumeTieredDeltaLateUsageCrossingTierBooksOnCurrentPeriod(t *testing.T) {
	// Given:
	// - a volume-tiered unit price and an unchanged intermediate run
	// When:
	// - late usage later moves the cumulative snapshot into the next tier
	// Then:
	// - the tier repricing correction is booked on the current period
	t.Parallel()

	periods := deltaRatingTestPeriods()
	price := volumeUnitTierPrice()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: price,
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:                periods.period2,
				meteredQuantity:       15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
			{
				period:          periods.period3,
				meteredQuantity: 16,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period3),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
					{
						ChildUniqueReferenceID: "volume-tiered-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period3),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaFlatTierRepricing(t *testing.T) {
	// Given:
	// - a volume-tiered flat price and a prior flat tier already booked
	// When:
	// - the current cumulative snapshot moves to a different flat tier
	// Then:
	// - the prior flat tier is reversed and the new flat tier is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: volumeFlatTierPrice(),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeFlatPriceChildUniqueReferenceID,
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
					Amount: 100,
					Total:  100,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 7,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeFlatPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          150,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
					{
						ChildUniqueReferenceID: "volume-flat-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          100,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -100,
							Total:  -100,
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

func TestVolumeTieredDeltaFlatTierToUnitTierTransition(t *testing.T) {
	// Given:
	// - a volume-tiered price that starts with flat tiers and then switches to a unit tier
	// When:
	// - the current cumulative snapshot crosses from the flat tier to the unit tier
	// Then:
	// - the flat tier is reversed and the unit-tier cumulative line is booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: volumeFlatToUnitTierPrice(),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 5,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeFlatPriceChildUniqueReferenceID,
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
					Amount: 100,
					Total:  100,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 6,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "volume-flat-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          100,
						Quantity:               -1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -100,
							Total:  -100,
						},
					},
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               6,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 60,
							Total:  60,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -40,
					Total:  -40,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaFlatTierWithNoUsage(t *testing.T) {
	// Given:
	// - a volume-tiered price whose first tier is a flat price
	// When:
	// - delta rating rates zero metered usage
	// Then:
	// - the flat tier is still booked
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: volumeFlatTierPrice(),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 0,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeFlatPriceChildUniqueReferenceID,
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
					Amount: 100,
					Total:  100,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaUnitTierWithNoUsage(t *testing.T) {
	// Given:
	// - a volume-tiered price whose first tier is a unit price
	// When:
	// - delta rating rates zero metered usage
	// Then:
	// - no detailed lines are produced
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.VolumeTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(5),
					},
				},
			},
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

func TestVolumeTieredDeltaMinimumCommitmentOnlyOnFinalPhase(t *testing.T) {
	// Given:
	// - a volume-tiered price with minimum commitment
	// - the first run is partial and the second run reaches the service-period end
	// When:
	// - delta rating rates both snapshots
	// Then:
	// - minimum commitment is ignored for the partial run and booked only on the final run
	t.Parallel()

	periods := deltaRatingTestPeriods()
	price := *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(10),
				},
			},
		},
		Commitments: productcatalog.Commitments{
			MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(150)),
		},
	})

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: price,
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               10,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 100,
							Total:  100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 100,
					Total:  100,
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
						PerUnitAmount:          50,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							ChargesTotal: 50,
							Total:        50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					ChargesTotal: 50,
					Total:        50,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaMaximumSpendWithTierRepricing(t *testing.T) {
	// Given:
	// - a volume-tiered unit price with maximum spend
	// - a prior run already hit the maximum spend in the first tier
	// When:
	// - the current cumulative snapshot reprices into the next tier
	// Then:
	// - the current tier line and the max-spend-adjusted reversal produce the delta
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: volumeUnitTierPriceWithCommitments(productcatalog.Commitments{
			MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
		}),
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         150,
							DiscountsTotal: 50,
							Total:          100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         150,
					DiscountsTotal: 50,
					Total:          100,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 16,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
					{
						ChildUniqueReferenceID: "volume-tiered-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         -150,
							DiscountsTotal: -50,
							Total:          -100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         -70,
					DiscountsTotal: -50,
					Total:          -20,
				},
			},
		},
	})
}

func TestVolumeTieredDeltaUsageDiscountWithTierRepricing(t *testing.T) {
	// Given:
	// - a volume-tiered unit price with a usage discount
	// - a prior run already booked discounted first-tier usage
	// When:
	// - the current cumulative snapshot reprices into the next tier
	// Then:
	// - the repriced discounted quantity is booked and the prior tier is reversed
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: volumeUnitTierPrice(),
		discounts: productcatalog.Discounts{
			Usage: lo.ToPtr(productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(5),
			}),
		},
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 20,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 21,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.VolumeUnitPriceChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
					{
						ChildUniqueReferenceID: "volume-tiered-price#correction:detailed_line_id=phase-1-line-1",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
		},
	})
}

func volumeUnitTierPrice() productcatalog.Price {
	return volumeUnitTierPriceWithCommitments(productcatalog.Commitments{})
}

func volumeUnitTierPriceWithCommitments(commitments productcatalog.Commitments) productcatalog.Price {
	return *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(15)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(10),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(5),
				},
			},
		},
		Commitments: commitments,
	})
}

func volumeFlatTierPrice() productcatalog.Price {
	return *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromInt(100),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromInt(150),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(5),
				},
			},
		},
	})
}

func volumeFlatToUnitTierPrice() productcatalog.Price {
	return *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(5)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					Amount: alpacadecimal.NewFromInt(100),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(10),
				},
			},
		},
	})
}
