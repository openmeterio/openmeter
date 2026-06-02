package mutator

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestApplyUsageDiscount(t *testing.T) {
	out, err := ApplyUsageDiscount(ApplyUsageDiscountInput{
		Usage: rating.Usage{
			Quantity:              alpacadecimal.NewFromInt(15),
			PreLinePeriodQuantity: alpacadecimal.NewFromInt(5),
		},
		RateCardDiscounts: billing.Discounts{
			Usage: &billing.UsageDiscount{
				UsageDiscount: productcatalog.UsageDiscount{
					Quantity: alpacadecimal.NewFromInt(10),
				},
				CorrelationID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, float64(10), out.Usage.Quantity.InexactFloat64())
	require.Equal(t, float64(0), out.Usage.PreLinePeriodQuantity.InexactFloat64())

	require.Len(t, out.StandardLineDiscounts.Usage, 1)
	usageDiscount := out.StandardLineDiscounts.Usage[0]
	require.Equal(t, "rateCardDiscount/correlationID=01ARZ3NDEKTSV4RRFFQ69G5FAV", lo.FromPtr(usageDiscount.ChildUniqueReferenceID))
	require.Equal(t, float64(5), usageDiscount.Quantity.InexactFloat64())
	require.Equal(t, float64(5), lo.FromPtr(usageDiscount.PreLinePeriodQuantity).InexactFloat64())
}

func TestApplyUsageDiscountWhenDiscountAlreadyConsumed(t *testing.T) {
	out, err := ApplyUsageDiscount(ApplyUsageDiscountInput{
		Usage: rating.Usage{
			Quantity:              alpacadecimal.NewFromInt(15),
			PreLinePeriodQuantity: alpacadecimal.NewFromInt(20),
		},
		RateCardDiscounts: billing.Discounts{
			Usage: &billing.UsageDiscount{
				UsageDiscount: productcatalog.UsageDiscount{
					Quantity: alpacadecimal.NewFromInt(10),
				},
				CorrelationID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, float64(15), out.Usage.Quantity.InexactFloat64())
	require.Equal(t, float64(10), out.Usage.PreLinePeriodQuantity.InexactFloat64())
	require.Empty(t, out.StandardLineDiscounts.Usage)
}
