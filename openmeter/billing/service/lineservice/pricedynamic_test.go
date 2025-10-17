package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestDynamicPriceCalculation(t *testing.T) {
	t.Run("dynamic price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("dynamic price, no usage, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
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
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
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
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
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
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(50), // 50 * 1.33333 = 66.67
				PreLinePeriodQty: alpacadecimal.NewFromFloat(70), // 70 * 1.33333 = 93.33
				// Total: 160
			},
			previousBilledAmount: alpacadecimal.NewFromFloat(93.33),
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(66.67),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(60),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
				},
			},
		})
	})
}
