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

var DecimalOne = alpacadecimal.NewFromInt(1)

func TestDynamicPriceCalculation(t *testing.T) {
	t.Run("dynamic price, no usage", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("dynamic price, no usage, min spend set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					Period:                 lo.ToPtr(testutil.TestFullPeriod.ToClosedPeriod()),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	// Min spend is always billed in arrears => we are not billing it in advance
	t.Run("no usage, not the last line in period, min spend set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{},
		})
	})

	// Min spend is always billed in arrears => we are billing it for the last line
	t.Run("no usage, last line in period, min spend set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: DecimalOne,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					Period:                 lo.ToPtr(testutil.TestFullPeriod.ToClosedPeriod()),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	// Usage is billed regardless of line position
	t.Run("usage present", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(13.33),
						Total:  alpacadecimal.NewFromFloat(13.33),
					},
				},
			},
		})
	})

	t.Run("usage present, mid line", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(13.33),
						Total:  alpacadecimal.NewFromFloat(13.33),
					},
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(13.33),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(13.33),
						Total:  alpacadecimal.NewFromFloat(13.33),
					},
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
				Multiplier: alpacadecimal.NewFromFloat(1.33333),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(50), // 50 * 1.33333 = 66.67
				PreLinePeriodQty: alpacadecimal.NewFromFloat(70), // 70 * 1.33333 = 93.33
				// Total: 160
			},
			PreviousBilledAmount: alpacadecimal.NewFromFloat(93.33),
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(66.67),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID,
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
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromFloat(66.67),
						DiscountsTotal: alpacadecimal.NewFromFloat(60),
						Total:          alpacadecimal.NewFromFloat(6.67),
					},
				},
			},
		})
	})
}
