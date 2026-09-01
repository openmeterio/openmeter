package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestListCreditTransactionsFundedBalanceCoalescesSameEffectiveTime(t *testing.T) {
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

func TestListCreditTransactionsFundedBalanceUsesEffectiveTimes(t *testing.T) {
	// Each scenario issues one future-effective credit purchase, then lists its
	// funded history both before and after the purchase becomes effective.
	type expectedTransaction struct {
		bookedAfter time.Duration
		amount      int64
		before      int64
		after       int64
	}

	tests := []struct {
		name          string
		advanceAmount int64
		before        []expectedTransaction
		after         []expectedTransaction
	}{
		{
			name:   "ordinary future issuance",
			before: []expectedTransaction{},
			after: []expectedTransaction{
				{bookedAfter: time.Hour, amount: 100, before: 0, after: 100},
			},
		},
		{
			name:          "future issuance partially covers advance",
			advanceAmount: 40,
			before: []expectedTransaction{
				{amount: 40, before: -40, after: 0},
			},
			after: []expectedTransaction{
				{bookedAfter: time.Hour, amount: 60, before: 0, after: 60},
				{amount: 40, before: -40, after: 0},
			},
		},
		{
			name:          "future issuance fully covers advance",
			advanceAmount: 100,
			before: []expectedTransaction{
				{amount: 100, before: -100, after: 0},
			},
			after: []expectedTransaction{
				{amount: 100, before: -100, after: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)

			// given:
			// - a customer may have an outstanding advance when a future-effective
			//   credit purchase is funded
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
			effectiveAt := fundedAt.Add(time.Hour)
			charge := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(100), env.Currency, &effectiveAt)

			// when:
			// - funded history is listed before the scheduled issuance is effective
			before, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Type:          lo.ToPtr(CreditTransactionTypeFunded),
				Currency:      &env.Currency,
				AsOf:          &fundedAt,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - only the advance attribution that already affected balance is visible
			require.Len(t, before.Items, len(tt.before))
			for i, want := range tt.before {
				item := before.Items[i]
				require.Equal(t, charge.ID, item.ID.ID)
				require.True(t, fundedAt.Add(want.bookedAfter).Equal(item.BookedAt))
				require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
				require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
				require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())
			}

			// when:
			// - funded history is listed after the scheduled issuance is effective
			afterEffectiveAt := effectiveAt.Add(time.Second)
			after, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID:    env.CustomerID,
				Limit:         10,
				Type:          lo.ToPtr(CreditTransactionTypeFunded),
				Currency:      &env.Currency,
				AsOf:          &afterEffectiveAt,
				FeatureFilter: AllFeatureFilter(),
			})
			require.NoError(t, err)

			// then:
			// - each distinct effective time has its own correctly ordered balance line
			require.Len(t, after.Items, len(tt.after))
			for i, want := range tt.after {
				item := after.Items[i]
				require.Equal(t, charge.ID, item.ID.ID)
				require.True(t, fundedAt.Add(want.bookedAfter).Equal(item.BookedAt))
				require.Equal(t, float64(want.amount), item.Amount.InexactFloat64())
				require.Equal(t, float64(want.before), item.Balance.Before.InexactFloat64())
				require.Equal(t, float64(want.after), item.Balance.After.InexactFloat64())
			}
		})
	}
}
