package pricer

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestAddDiscountForOverage(t *testing.T) {
	currency, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	l := DetailedLine{
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		Quantity:      alpacadecimal.NewFromFloat(10),
	}

	t.Run("no overage", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, l, lineWithDiscount)
	})

	// currency rounding
	t.Run("no overage", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000.001),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000.001),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, l, lineWithDiscount)
	})

	t.Run("overage, rounding", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000.001),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000.01123),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, DetailedLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(0.01),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
		}, lineWithDiscount)
	})

	t.Run("overage and some valid charges", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9600),
			// Total $10000 => $500 discount
			Currency: currency,
		})

		require.Equal(t, DetailedLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(600),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
		}, lineWithDiscount)
	})

	t.Run("overage 100% discount", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(10000),
			// Total $10000 => $1000 discount
			Currency: currency,
		})

		require.Equal(t, DetailedLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(1000),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
		}, lineWithDiscount)
	})

	t.Run("overage and 100% discount when hugely over the max spend", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(AddDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(20000),
			// Total $10000 => $1000 discount
			Currency: currency,
		})

		require.Equal(t, DetailedLine{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(1000),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
		}, lineWithDiscount)
	})
}
