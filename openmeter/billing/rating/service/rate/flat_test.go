package rate_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestFlatLineCalculation(t *testing.T) {
	// Flat price tests
	t.Run("flat price, in advance", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			// Note: this is just the qty of the line, no feature lookup is done
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InAdvancePaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(100),
						Total:  alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	t.Run("flat price, in arrears", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			// Note: this is just the qty of the line, no feature lookup is done
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(100),
						Total:  alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	t.Run("flat price, in advance, mid period", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			Expect: rating.DetailedLines{},
		})
	})

	t.Run("flat price, in arrears, single period line", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Expect: rating.DetailedLines{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(100),
						Total:  alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	t.Run("flat price, in arrears,  mid period line", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Expect:   rating.DetailedLines{}, // It will be billed in the last period
		})
	})

	t.Run("flat price, in arrears, last period line", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Expect: rating.DetailedLines{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(100),
						Total:  alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})
}
