package customerbalance

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestListCreditTransactionsTypeFilterPreservesHistoricalBalances(t *testing.T) {
	env := newTestEnv(t)
	issuedAt := clock.Now()
	clock.FreezeTime(issuedAt)
	defer clock.UnFreeze()

	// given: funding, spending, expiry, and voids interleave in one credit history.
	env.createPromotionalCreditFunding(t, issuedAt, alpacadecimal.NewFromInt(100), issuedAt.Add(5*time.Hour))
	consumedAt := issuedAt.Add(time.Hour)
	clock.FreezeTime(consumedAt.Add(-time.Minute))
	charge := env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode,
		timeutil.ClosedPeriod{From: consumedAt, To: consumedAt}, env.Currency)
	clock.FreezeTime(consumedAt.Add(time.Second))
	env.advanceFlatFeeCharge(t, charge)
	env.createPromotionalCreditFunding(t, issuedAt.Add(2*time.Hour), alpacadecimal.NewFromInt(50), issuedAt.Add(7*time.Hour))
	clock.FreezeTime(issuedAt.Add(150 * time.Minute))
	firstVoid := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(20), env.Currency, nil)
	clock.FreezeTime(issuedAt.Add(3 * time.Hour))
	env.voidFundedCreditGrant(t, firstVoid)
	clock.FreezeTime(issuedAt.Add(4 * time.Hour))
	secondVoid := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(10), env.Currency, nil)
	clock.FreezeTime(issuedAt.Add(6 * time.Hour))
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(40), env.Currency, nil)
	clock.FreezeTime(issuedAt.Add(8 * time.Hour))
	env.voidFundedCreditGrant(t, secondVoid)

	input := ListCreditTransactionsInput{CustomerID: env.CustomerID, Limit: 20, Currency: &env.Currency, FeatureFilter: AllFeatureFilter()}
	unfiltered, err := env.Service.ListCreditTransactions(t.Context(), input)
	require.NoError(t, err)
	require.Len(t, unfiltered.Items, 10)
	oldest := unfiltered.Items[len(unfiltered.Items)-1]
	require.Equal(t, float64(0), oldest.Balance.Before.InexactFloat64())
	require.Equal(t, float64(100), oldest.Balance.After.InexactFloat64())

	counter := &countingBoundaryBalanceQuerier{BalanceQuerier: env.Service.BalanceQuerier}
	env.Service.BalanceQuerier = counter
	for _, transactionType := range []CreditTransactionType{CreditTransactionTypeFunded, CreditTransactionTypeConsumed, CreditTransactionTypeExpired, CreditTransactionTypeVoided} {
		t.Run(string(transactionType), func(t *testing.T) {
			// when: a type filter omits the other movements between selected rows.
			input.Type = &transactionType
			input.Limit = 20
			input.After, input.Before = nil, nil
			counter.calls = 0
			filtered, err := env.Service.ListCreditTransactions(t.Context(), input)
			require.NoError(t, err)
			expected := lo.Filter(unfiltered.Items, func(item CreditTransaction, _ int) bool { return item.Type == transactionType })
			require.Len(t, filtered.Items, len(expected))
			for idx, item := range filtered.Items {
				requireCreditTransactionBalanceMatches(t, expected[idx], item)
			}
			require.Equal(t, 1, counter.calls)

			// then: forward and backward cursor pages preserve those same balances.
			input.Limit = 1
			for idx, item := range expected {
				page, err := env.Service.ListCreditTransactions(t.Context(), input)
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				requireCreditTransactionBalanceMatches(t, item, page.Items[0])
				if idx+1 < len(expected) {
					require.NotNil(t, page.NextCursor)
					input.After = page.NextCursor
				}
			}
			input.After = nil
			for idx := len(expected) - 1; idx > 0; idx-- {
				input.Before = lo.ToPtr(creditTransactionCursor(expected[idx]))
				page, err := env.Service.ListCreditTransactions(t.Context(), input)
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				requireCreditTransactionBalanceMatches(t, expected[idx-1], page.Items[0])
			}
		})
	}
}

func TestListCreditTransactionsSharedTerminalTimePreservesBalancesAcrossPages(t *testing.T) {
	for _, transactionType := range []CreditTransactionType{CreditTransactionTypeExpired, CreditTransactionTypeVoided} {
		t.Run(string(transactionType), func(t *testing.T) {
			env := newTestEnv(t)
			issuedAt := clock.Now()
			terminalAt := issuedAt.Add(5 * time.Hour)
			clock.FreezeTime(issuedAt)
			defer clock.UnFreeze()

			// given: three grants expire or are voided at the same effective time.
			var purchases []creditpurchase.Charge
			for idx, amount := range []int64{10, 20, 30} {
				fundedAt := issuedAt.Add(time.Duration(idx) * time.Hour)
				clock.FreezeTime(fundedAt)
				if transactionType == CreditTransactionTypeExpired {
					purchases = append(purchases, env.createPromotionalCreditFunding(t, fundedAt, alpacadecimal.NewFromInt(amount), terminalAt))
				} else {
					purchases = append(purchases, env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(amount), env.Currency, nil))
				}
			}
			clock.FreezeTime(terminalAt)
			if transactionType == CreditTransactionTypeVoided {
				for _, purchase := range purchases {
					env.voidFundedCreditGrant(t, purchase)
				}
			}

			// when: the selected terminal rows share a persisted timestamp boundary.
			input := ListCreditTransactionsInput{CustomerID: env.CustomerID, Limit: 10, Currency: &env.Currency, FeatureFilter: AllFeatureFilter()}
			all, err := env.Service.ListCreditTransactions(t.Context(), input)
			require.NoError(t, err)
			require.Len(t, all.Items, 6)
			expected := all.Items[:3]
			require.Equal(t, float64(0), expected[0].Balance.After.InexactFloat64())
			require.Equal(t, float64(60), expected[2].Balance.Before.InexactFloat64())
			input.Type = &transactionType
			filtered, err := env.Service.ListCreditTransactions(t.Context(), input)
			require.NoError(t, err)
			require.Len(t, filtered.Items, 3)
			for idx, item := range filtered.Items {
				requireCreditTransactionBalanceMatches(t, expected[idx], item)
			}

			// then: even a page starting inside the shared timestamp retains sibling impacts.
			input.Limit = 1
			for _, filter := range []*CreditTransactionType{nil, &transactionType} {
				input.Type = filter
				input.Before = nil
				input.After = lo.ToPtr(creditTransactionCursor(expected[0]))
				page, err := env.Service.ListCreditTransactions(t.Context(), input)
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				requireCreditTransactionBalanceMatches(t, expected[1], page.Items[0])
				input.After = nil
				input.Before = lo.ToPtr(creditTransactionCursor(expected[2]))
				page, err = env.Service.ListCreditTransactions(t.Context(), input)
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				requireCreditTransactionBalanceMatches(t, expected[1], page.Items[0])
			}
		})
	}
}

func TestListCreditTransactionsMixedTypesAtSameTimePreserveBalances(t *testing.T) {
	env := newTestEnv(t)
	issuedAt := clock.Now()
	terminalAt := issuedAt.Add(time.Hour)
	clock.FreezeTime(issuedAt)
	defer clock.UnFreeze()

	// given: expiry, a void, and new funding all affect the same booked timestamp.
	env.createPromotionalCreditFunding(t, issuedAt, alpacadecimal.NewFromInt(10), terminalAt)
	toVoid := env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(20), env.Currency, nil)
	clock.FreezeTime(terminalAt)
	env.voidFundedCreditGrant(t, toVoid)
	env.createPromotionalCreditGrant(t, alpacadecimal.NewFromInt(30), env.Currency, nil)
	input := ListCreditTransactionsInput{CustomerID: env.CustomerID, Limit: 10, Currency: &env.Currency, FeatureFilter: AllFeatureFilter()}
	all, err := env.Service.ListCreditTransactions(t.Context(), input)
	require.NoError(t, err)
	require.Len(t, all.Items, 5)

	// when: each kind is selected separately, including terminal net projections.
	for _, transactionType := range []CreditTransactionType{CreditTransactionTypeFunded, CreditTransactionTypeExpired, CreditTransactionTypeVoided} {
		input.Type = &transactionType
		filtered, err := env.Service.ListCreditTransactions(t.Context(), input)
		require.NoError(t, err)
		expected := lo.Filter(all.Items, func(item CreditTransaction, _ int) bool { return item.Type == transactionType })
		require.Len(t, filtered.Items, len(expected))
		for idx, item := range filtered.Items {
			// then: a row's boundary is independent of which neighboring kinds are visible.
			requireCreditTransactionBalanceMatches(t, expected[idx], item)
			if transactionType != CreditTransactionTypeFunded {
				require.Equal(t, float64(30), item.Balance.After.InexactFloat64())
			}
		}
	}
}

func requireCreditTransactionBalanceMatches(t *testing.T, expected, actual CreditTransaction) {
	t.Helper()
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Balance.Before.InexactFloat64(), actual.Balance.Before.InexactFloat64())
	require.Equal(t, expected.Balance.After.InexactFloat64(), actual.Balance.After.InexactFloat64())
}

type countingBoundaryBalanceQuerier struct {
	ledger.BalanceQuerier
	calls int
}

func (q *countingBoundaryBalanceQuerier) GetBalancesAtBoundaries(ctx context.Context, input ledger.GetBalancesAtBoundariesInput) ([]ledger.Balance, error) {
	q.calls++
	return q.BalanceQuerier.GetBalancesAtBoundaries(ctx, input)
}

// voidFundedCreditGrant follows the grant void path: book the ledger forfeiture
// and mark the charge in the same database transaction.
func (e *testEnv) voidFundedCreditGrant(t *testing.T, charge creditpurchase.Charge) {
	t.Helper()
	_, err := transaction.Run(t.Context(), enttx.NewCreator(e.DB), func(ctx context.Context) (creditpurchase.ChargeBase, error) {
		result, err := e.CreditVoidService.VoidCreditPurchase(ctx, creditvoid.VoidCreditPurchaseInput{
			CustomerID: e.CustomerID,
			ChargeID:   charge.ID,
			Currency:   charge.Intent.Currency.GetCode(),
		})
		if err != nil {
			return creditpurchase.ChargeBase{}, err
		}
		return e.creditPurchase.MarkVoided(ctx, creditpurchase.MarkVoidedInput{
			ChargeID: charge.GetChargeID(),
			VoidedAt: result.VoidedAt,
		})
	})
	require.NoError(t, err)
}
