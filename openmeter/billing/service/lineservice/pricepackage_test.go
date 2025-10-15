package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestPackagePriceCalculation(t *testing.T) {
	t.Run("package price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("package price, no usage, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
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
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
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
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
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
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line, in first package", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line, usage starts in a mid period split line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line, overflow into second package", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(17),
				LinePeriodQty:    alpacadecimal.NewFromFloat(5),
			},
			// The first in period line has already billed for usage [0..17]
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
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
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.PackagePrice{
				QuantityPerPackage: alpacadecimal.NewFromFloat(10),
				Amount:             alpacadecimal.NewFromFloat(15),
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(50), // 5 * 15 = 75
				PreLinePeriodQty: alpacadecimal.NewFromFloat(60), // 6 * 15 = 90
				// Total: 165
			},
			previousBilledAmount: alpacadecimal.NewFromFloat(90),
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(15),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: UsageChildUniqueReferenceID,
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
				},
			},
		})
	})
}

func TestGetNumberOfPackages(t *testing.T) {
	p := packagePricer{}

	require.Equal(t, float64(0), p.getNumberOfPackages(alpacadecimal.NewFromFloat(0), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.getNumberOfPackages(alpacadecimal.NewFromFloat(1), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.getNumberOfPackages(alpacadecimal.NewFromFloat(9), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(1), p.getNumberOfPackages(alpacadecimal.NewFromFloat(10), alpacadecimal.NewFromFloat(10)).InexactFloat64())
	require.Equal(t, float64(2), p.getNumberOfPackages(alpacadecimal.NewFromFloat(11), alpacadecimal.NewFromFloat(10)).InexactFloat64())
}
