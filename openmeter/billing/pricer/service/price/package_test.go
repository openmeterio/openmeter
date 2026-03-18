package price_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/price"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestPackagePriceCalculation(t *testing.T) {
	t.Run("package price, no usage", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: pricer.DetailedLines{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("package price, no usage, min spend set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.MinSpendChildUniqueReferenceID,
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
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: pricer.DetailedLines{},
		})
	})

	// Min spend is always billed in arrears => we are billing it for the last line
	t.Run("no usage, last line in period, min spend set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.MinSpendChildUniqueReferenceID,
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
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})

	t.Run("usage present, mid line, in first package", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})

	t.Run("usage present, mid line, usage starts in a mid period split line", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})

	t.Run("usage present, mid line, overflow into second package", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(17),
				LinePeriodQty:    alpacadecimal.NewFromFloat(5),
			},
			// The first in period line has already billed for usage [0..17]
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(50), // 5 * 15 = 75
				PreLinePeriodQty: alpacadecimal.NewFromFloat(60), // 6 * 15 = 90
				// Total: 165
			},
			PreviousBilledAmount: alpacadecimal.NewFromFloat(90),
			Expect: pricer.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(65),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromFloat(75),
						DiscountsTotal: alpacadecimal.NewFromFloat(65),
						Total:          alpacadecimal.NewFromFloat(10),
					},
				},
			},
		})
	})
}

func TestGetNumberOfPackages(t *testing.T) {
	p := price.Package{}

	require.Equal(t, float64(0), p.GetNumberOfPackages(alpacadecimal.NewFromFloat(0), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.GetNumberOfPackages(alpacadecimal.NewFromFloat(1), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.GetNumberOfPackages(alpacadecimal.NewFromFloat(9), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.GetNumberOfPackages(alpacadecimal.NewFromFloat(10), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(2), p.GetNumberOfPackages(alpacadecimal.NewFromFloat(11), alpacadecimal.NewFromFloat(10)).InexactFloat64())
}
