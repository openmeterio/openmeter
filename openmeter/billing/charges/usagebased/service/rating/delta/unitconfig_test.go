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

// TestUnitConfigMultiplyAppliedToRatedQuantity asserts that
// UnitPrice(1) + UnitConfig{multiply, m} rates raw qty × m as the billable
// quantity. This is the v3 equivalent of v1 DynamicPrice{multiplier: m} for
// the same raw qty (line totals match; only ChildUniqueReferenceID differs
// because v3 surfaces as a unit price).
func TestUnitConfigMultiplyAppliedToRatedQuantity(t *testing.T) {
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
		unitConfig: &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationMultiply,
			ConversionFactor: alpacadecimal.NewFromFloat(1.5),
		},
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 10,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          1,
						Quantity:               15,
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

// TestUnitConfigDivideCeilingAppliedToRatedQuantity asserts that
// UnitPrice(amount) + UnitConfig{divide, qty, ceiling} rates raw qty ÷ qty
// (rounded up) as the billable quantity. v3 equivalent of v1
// PackagePrice{amount, qty} for the same raw qty.
func TestUnitConfigDivideCeilingAppliedToRatedQuantity(t *testing.T) {
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		unitConfig: &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationDivide,
			ConversionFactor: alpacadecimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(productcatalog.UnitConfigRoundingModeCeiling),
		},
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1247,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
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

// TestUnitConfigDivideCeilingCumulativeNoDoubleBilling asserts a key
// correctness property: applying UnitConfig{divide, ceiling} to cumulative
// raw quantity (not the per-run diff), then letting the delta engine's
// existing cumulative-minus-prior subtraction run, prevents double-billing
// when the customer adds raw usage that does not cross a new package
// boundary.
//
// Run 1: raw=1247 → invoiced=ceil(1.247)=2 packages → bill 2 packages.
// Run 2: raw=1500 → invoiced=ceil(1.500)=2 packages → same cumulative, no
// new line.
//
// Wrong design (apply UnitConfig to the diff) would charge:
// ceil((1500-1247)/1000) = ceil(0.253) = 1 additional package.
// Right design (apply to cumulative, diff in invoiced space) charges 0.
func TestUnitConfigDivideCeilingCumulativeNoDoubleBilling(t *testing.T) {
	t.Parallel()

	periods := deltaRatingTestPeriods()

	runDeltaRatingTestCase(t, deltaRatingTestCase{
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		unitConfig: &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationDivide,
			ConversionFactor: alpacadecimal.NewFromInt(1000),
			Rounding:         lo.ToPtr(productcatalog.UnitConfigRoundingModeCeiling),
		},
		phases: []deltaRatingPhase{
			{
				period:          periods.period1,
				meteredQuantity: 1247,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
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
			{
				period:                periods.period2,
				meteredQuantity:       1500,
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:        ratingtestutils.ExpectedTotals{},
			},
		},
	})
}
