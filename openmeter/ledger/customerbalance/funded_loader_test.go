package customerbalance

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestListCreditTransactionsFundedBalanceIncludesAdvanceAttribution(t *testing.T) {
	tests := []struct {
		name          string
		advanceAmount int64
		wantBefore    int64
		wantAfter     int64
	}{
		{
			name:       "ordinary issuance",
			wantBefore: 0,
			wantAfter:  100,
		},
		{
			name:          "partially covered advance",
			advanceAmount: 40,
			wantBefore:    -40,
			wantAfter:     60,
		},
		{
			name:          "fully covered advance without issuance",
			advanceAmount: 100,
			wantBefore:    -100,
			wantAfter:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)

			// given:
			// - the customer may already have settled advance usage
			if tt.advanceAmount > 0 {
				servicePeriod := env.sp()
				charge := env.createFlatFeeCharge(
					t,
					alpacadecimal.NewFromInt(tt.advanceAmount),
					productcatalog.CreditOnlySettlementMode,
					servicePeriod,
				)
				env.passTimeAfterServicePeriod(t, servicePeriod)
				env.advanceFlatFeeCharge(t, charge)
			}

			fundedAt := clock.Now()
			env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, nil)

			// when:
			// - merged transaction history reconstructs the balance around the purchase
			result, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Currency:      &env.Currency,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - the full purchase is visible and its before/after balance includes both
			//   issued FBO and cleared advance receivable value
			expectedItems := 1
			if tt.advanceAmount > 0 {
				expectedItems++
			}
			require.Len(t, result.Items, expectedItems)
			item := result.Items[0]
			require.Equal(t, CreditTransactionTypeFunded, item.Type)
			require.True(t, fundedAt.Equal(item.BookedAt))
			require.Equal(t, float64(100), item.Amount.InexactFloat64())
			require.Equal(t, float64(tt.wantBefore), item.Balance.Before.InexactFloat64())
			require.Equal(t, float64(tt.wantAfter), item.Balance.After.InexactFloat64())

			if tt.advanceAmount > 0 {
				advance := result.Items[1]
				require.Equal(t, CreditTransactionTypeConsumed, advance.Type)
				require.Equal(t, float64(-tt.advanceAmount), advance.Amount.InexactFloat64())
				require.Equal(t, float64(0), advance.Balance.Before.InexactFloat64())
				require.Equal(t, float64(-tt.advanceAmount), advance.Balance.After.InexactFloat64())
			}
		})
	}
}
