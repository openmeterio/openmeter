package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestFlatLineCalculation(t *testing.T) {
	// Flat price tests
	t.Run("flat price no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
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

	t.Run("flat price, in advance, usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
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

	t.Run("flat price, in advance, usage present, mid period", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("flat price, in arrears, usage present, single period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
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

	t.Run("flat price, in arrears, usage present, mid period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{}, // It will be billed in the last period
		})
	})

	t.Run("flat price, in arrears, usage present, last period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
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
}
