package detailedline

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
)

func TestAmountDiscountsValidateAllowsSignedAmounts(t *testing.T) {
	t.Parallel()

	discounts := AmountDiscounts{
		{
			ChildUniqueReferenceID: "maximum-spend",
			Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
			Amount:                 alpacadecimal.NewFromInt(10),
			RoundingAmount:         alpacadecimal.NewFromFloat(0.01),
		},
		{
			ChildUniqueReferenceID: "maximum-spend-reversal",
			Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
			Amount:                 alpacadecimal.NewFromInt(-4),
			RoundingAmount:         alpacadecimal.NewFromFloat(-0.01),
		},
	}

	require.NoError(t, discounts.Validate())
	require.Equal(
		t,
		float64(6),
		discounts.SumAmount(currenciestestutils.NewFiatCurrency(t, "USD")).InexactFloat64(),
	)
}

func TestAmountDiscountsValidateRequiresStableReferenceAndReason(t *testing.T) {
	t.Parallel()

	err := (AmountDiscounts{{}}).Validate()
	require.ErrorContains(t, err, "child unique reference id is required")
	require.ErrorContains(t, err, "invalid discount type")
}

func TestAmountDiscountsClonePreservesNilAndEmptyCollections(t *testing.T) {
	t.Parallel()

	require.Nil(t, AmountDiscounts(nil).Clone())

	cloned := (AmountDiscounts{}).Clone()
	require.NotNil(t, cloned)
	require.Empty(t, cloned)
}

func TestAmountDiscountsArithmetic(t *testing.T) {
	t.Parallel()

	percentage := billing.NewDiscountReasonFrom(billing.PercentageDiscount{CorrelationID: "pct"})
	maximumSpend := billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{})

	discount := func(id string, reason billing.DiscountReason, amount, rounding float64) AmountDiscount {
		return AmountDiscount{
			ChildUniqueReferenceID: id,
			Reason:                 reason,
			Amount:                 alpacadecimal.NewFromFloat(amount),
			RoundingAmount:         alpacadecimal.NewFromFloat(rounding),
		}
	}

	t.Run("sub nets matched entries and keeps current-only entries", func(t *testing.T) {
		current := AmountDiscounts{
			discount("pct", percentage, 12.5, 0.03),
			discount("max", maximumSpend, 4, 0),
		}
		previous := AmountDiscounts{
			discount("pct", percentage, 10, 0.01),
		}

		require.Equal(t, AmountDiscounts{
			discount("pct", percentage, 2.5, 0.02),
			discount("max", maximumSpend, 4, 0),
		}, current.Sub(previous))
	})

	t.Run("sub reverses previous-only entries with their reason", func(t *testing.T) {
		current := AmountDiscounts{
			discount("pct", percentage, 12.5, 0),
		}
		previous := AmountDiscounts{
			discount("pct", percentage, 10, 0),
			discount("max", maximumSpend, 4, 0.01),
		}

		require.Equal(t, AmountDiscounts{
			discount("pct", percentage, 2.5, 0),
			discount("max", maximumSpend, -4, -0.01),
		}, current.Sub(previous))
	})

	t.Run("sub drops entries that net to zero", func(t *testing.T) {
		current := AmountDiscounts{discount("pct", percentage, 10, 0.01)}

		require.Empty(t, current.Sub(current))
	})

	t.Run("sub keeps rounding-only deltas", func(t *testing.T) {
		current := AmountDiscounts{discount("pct", percentage, 10, 0.03)}
		previous := AmountDiscounts{discount("pct", percentage, 10, 0.01)}

		require.Equal(t, AmountDiscounts{discount("pct", percentage, 0, 0.02)}, current.Sub(previous))
	})

	t.Run("add sums matched entries and appends new ones in order", func(t *testing.T) {
		a := AmountDiscounts{discount("pct", percentage, 1, 0)}
		b := AmountDiscounts{
			discount("max", maximumSpend, 2, 0),
			discount("pct", percentage, 3, 0.01),
		}

		require.Equal(t, AmountDiscounts{
			discount("pct", percentage, 4, 0.01),
			discount("max", maximumSpend, 2, 0),
		}, a.Add(b))
	})

	t.Run("neg negates amount and rounding without mutating the receiver", func(t *testing.T) {
		original := AmountDiscounts{discount("pct", percentage, 10, 0.01)}
		negated := original.Neg()

		require.Equal(t, AmountDiscounts{discount("pct", percentage, -10, -0.01)}, negated)
		require.Equal(t, AmountDiscounts{discount("pct", percentage, 10, 0.01)}, original)
	})
}
