package rate_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// TestUnitConfigRating exercises the unit_config conversion end-to-end through the
// real rating service and unit pricer.
func TestUnitConfigRating(t *testing.T) {
	unitPrice := *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(10)})

	divideCeiling := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	t.Run("flag on bills the converted, rounded quantity", func(t *testing.T) {
		// 1400 raw / 1000 = 1.4, ceiling -> 2 billed units at 10 = 20.
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price:             unitPrice,
			UnitConfig:        divideCeiling,
			UnitConfigEnabled: true,
			LineMode:          testutil.SinglePerPeriodLineMode,
			Usage:             testutil.FeatureUsageResponse{LinePeriodQty: alpacadecimal.NewFromFloat(1400)},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(2),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(20),
						Total:  alpacadecimal.NewFromFloat(20),
					},
				},
			},
		})
	})

	t.Run("progressive split rounds on cumulative endpoints", func(t *testing.T) {
		// Mid-period split with a non-zero pre-line quantity: cumulative 1400 -> 2700,
		// divide by 1000, ceiling. start' = ceil(1.4) = 2, end' = ceil(2.7) = 3, so the
		// billed delta is 1 -- NOT per-line ceil(1300/1000) = 2 -- at $10 = $10.
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price:             unitPrice,
			UnitConfig:        divideCeiling,
			UnitConfigEnabled: true,
			LineMode:          testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(1400),
				LinePeriodQty:    alpacadecimal.NewFromFloat(1300),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(10),
						Total:  alpacadecimal.NewFromFloat(10),
					},
				},
			},
		})
	})

	t.Run("flag off bills the raw quantity (parity with today)", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price:             unitPrice,
			UnitConfig:        divideCeiling,
			UnitConfigEnabled: false,
			LineMode:          testutil.SinglePerPeriodLineMode,
			Usage:             testutil.FeatureUsageResponse{LinePeriodQty: alpacadecimal.NewFromFloat(1400)},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(1400),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(14000),
						Total:  alpacadecimal.NewFromFloat(14000),
					},
				},
			},
		})
	})

	t.Run("graduated tiers operate in converted units", func(t *testing.T) {
		// Meter is in bytes, tiers are authored in GB. 7400 bytes / 1000 = 7.4 -> ceil 8 GB.
		// Graduated over [0, 8]: tier 1 [0,5] @ 1 = 5, tier 2 (open) [5,8] @ 0.5 = 1.5.
		gradPrice := *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
					UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(1)},
				},
				{
					UnitPrice: &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(0.5)},
				},
			},
		})

		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price:             gradPrice,
			UnitConfig:        divideCeiling,
			UnitConfigEnabled: true,
			LineMode:          testutil.SinglePerPeriodLineMode,
			Usage:             testutil.FeatureUsageResponse{LinePeriodQty: alpacadecimal.NewFromFloat(7400)},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(5),
						Total:  alpacadecimal.NewFromFloat(5),
					},
				},
				{
					Name:                   "feature: usage price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(0.5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "graduated-tiered-2-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(1.5),
						Total:  alpacadecimal.NewFromFloat(1.5),
					},
				},
			},
		})
	})

	t.Run("convert then round then discount: usage discount applies to the rounded quantity", func(t *testing.T) {
		// 1400 / 1000 = 1.4 -> ceil 2; a 0.5 (converted-unit) usage discount -> 1.5 at 10 = 15.
		// Deliberately not ceil(1.4 - 0.5) = 1: the discount is against the rounded, displayed quantity.
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price:             unitPrice,
			UnitConfig:        divideCeiling,
			UnitConfigEnabled: true,
			LineMode:          testutil.SinglePerPeriodLineMode,
			Usage:             testutil.FeatureUsageResponse{LinePeriodQty: alpacadecimal.NewFromFloat(1400)},
			Discounts: billing.Discounts{
				Usage: &billing.UsageDiscount{
					UsageDiscount: productcatalog.UsageDiscount{
						Quantity: alpacadecimal.NewFromFloat(0.5),
					},
					CorrelationID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				},
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(1.5),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})
}
