package delta

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestGraduatedTieredDeltaDoesNotCarryExhaustedFreeTierToThirdInvoice(t *testing.T) {
	t.Parallel()

	// Given: one charge progressively invoiced three times, with the first ten
	// units free and every subsequent unit priced at $1.
	// When: cumulative usage reaches 5, 15, then 20 units, the first two invoices
	// already account for all ten free units.
	// Then: the third invoice contains only five additional paid units; an empty
	// free-tier delta has neither new usage nor an amount to present.
	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.Zero,
					},
				},
				{
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
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
						PerUnitAmount:          0,
						Quantity:               5,
					},
				},
			},
			{
				period:          periods.period2,
				meteredQuantity: 15,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          0,
						Quantity:               5,
					},
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
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
				period:          periods.period3,
				meteredQuantity: 20,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period3),
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
		},
	})
}
