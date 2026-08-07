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

	require.Nil(t, (AmountDiscounts(nil)).Clone())

	cloned := (AmountDiscounts{}).Clone()
	require.NotNil(t, cloned)
	require.Empty(t, cloned)
}
