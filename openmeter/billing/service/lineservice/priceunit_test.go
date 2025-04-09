package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestUnitPriceCalculation(t *testing.T) {
	t.Run("unit price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("unit price, no usage, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: MinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	// Min spend is always billed in arrears => we are not billing it in advance
	t.Run("no usage, not the last line in period, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// Min spend is always billed in arrears => we are billing it for the last line
	t.Run("no usage, last line in period, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: MinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	// Usage is billed regardless of line position
	t.Run("usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(5),
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
			},
			previousBilledAmount: alpacadecimal.NewFromFloat(7 * 10),
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: billing.NewLineDiscounts(
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount: alpacadecimal.NewFromFloat(20),
							LineDiscountBase: billing.LineDiscountBase{
								Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
								ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
								Reason:                 billing.LineDiscountReasonMaximumSpend,
							},
						},
						),
					),
				},
			},
		})
	})

	// Discount + max spend
	t.Run("usage present, 50% discount +max spend set and hit", func(t *testing.T) {
		discount50pct := billing.NewDiscountFrom(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(50),
		}).WithCorrelationID("discount-50pct")
		// PreLineUsage
		// 10*7*0.3 = 35
		// Current line usage:
		//   Amount: 10*20 = 200
		//   	Discount: 200*0.5 = -100
		//   Line total: 200-100 = 100
		//
		//   Total spend: 35+100 = 135
		//     Maximum spend: 100
		//
		//   Max spend discount: 135-100 = 35
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			discounts: []billing.Discount{
				discount50pct,
			},
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(20),
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
			},
			previousBilledAmount: alpacadecimal.NewFromFloat(3.5 * 10),
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(20), // 200
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: billing.NewLineDiscounts(
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount: alpacadecimal.NewFromFloat(100),
							LineDiscountBase: billing.LineDiscountBase{
								ChildUniqueReferenceID: lo.ToPtr("rateCardDiscount/correlationID=discount-50pct"),
								Reason:                 billing.LineDiscountReasonRatecardDiscount,
								SourceDiscount:         lo.ToPtr(discount50pct),
							},
						}),
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount: alpacadecimal.NewFromFloat(35),
							LineDiscountBase: billing.LineDiscountBase{
								Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
								ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
								Reason:                 billing.LineDiscountReasonMaximumSpend,
							},
						},
						),
					),
				},
			},
		})
	})

	t.Run("usage present, 33%+33%+34% discount, should yield 0", func(t *testing.T) {
		discount33pct := billing.NewDiscountFrom(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(33),
		})
		discount33pctV1 := discount33pct.WithCorrelationID("discount-33pct-1")
		discount33pctV2 := discount33pct.WithCorrelationID("discount-33pct-2")
		discount34pct := billing.NewDiscountFrom(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(34),
		}).WithCorrelationID("discount-34pct")
		// Current line usage:
		//   Amount: 0.01*1 = 0.01
		//   	Discount: 0.1*0.33 = 0.0
		// 	 	Discount: 0.1*0.33 = 0.0
		// 	 	Discount: 0.1*0.34 = 0.1 (rounding)
		//   Line total: 0.01-0.01*0.33-0.01*0.33-0.01*0.34 = 0
		//
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(0.01),
			}),
			discounts: []billing.Discount{
				discount33pctV1,
				discount33pctV2,
				discount34pct,
			},
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(0.01),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: billing.NewLineDiscounts(
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount: alpacadecimal.NewFromFloat(0),
							LineDiscountBase: billing.LineDiscountBase{
								ChildUniqueReferenceID: lo.ToPtr("rateCardDiscount/correlationID=discount-33pct-1"),
								Reason:                 billing.LineDiscountReasonRatecardDiscount,
								SourceDiscount:         lo.ToPtr(discount33pctV1),
							},
						}),
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount: alpacadecimal.NewFromFloat(0),
							LineDiscountBase: billing.LineDiscountBase{
								ChildUniqueReferenceID: lo.ToPtr("rateCardDiscount/correlationID=discount-33pct-2"),
								Reason:                 billing.LineDiscountReasonRatecardDiscount,
								SourceDiscount:         lo.ToPtr(discount33pctV2),
							},
						}),
						billing.NewLineDiscountFrom(billing.AmountLineDiscount{
							Amount:         alpacadecimal.NewFromFloat(0.0),
							RoundingAmount: alpacadecimal.NewFromFloat(0.01),
							LineDiscountBase: billing.LineDiscountBase{
								ChildUniqueReferenceID: lo.ToPtr("rateCardDiscount/correlationID=discount-34pct"),
								Reason:                 billing.LineDiscountReasonRatecardDiscount,
								SourceDiscount:         lo.ToPtr(discount34pct),
							},
						}),
					),
				},
			},
		})
	})
}
