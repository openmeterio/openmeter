package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestFlatLineCalculation(t *testing.T) {
	// Flat price tests
	t.Run("flat price, in advance", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			// Note: this is just the qty of the line, no feature lookup is done
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InAdvancePaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in arrears", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			// Note: this is just the qty of the line, no feature lookup is done
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in advance, mid period", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(1),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("flat price, in arrears, single period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in arrears,  mid period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			expect:   newDetailedLinesInput{}, // It will be billed in the last period
		})
	})

	t.Run("flat price, in arrears, last period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: lastInPeriodSplitLineMode,
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})
}
