package customerbalance

import (
	"cmp"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestListCreditTransactionsExpiredBreakage(t *testing.T) {
	issuedAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		plans           []expiredListingPlanSetup
		expectedExpired []expectedExpiredListingTransaction
	}{
		{
			name: "unused credit lists full expiry",
			plans: []expiredListingPlanSetup{
				{amount: 10, expiresAfter: 10 * time.Hour},
			},
			expectedExpired: []expectedExpiredListingTransaction{
				{expiresAfter: 10 * time.Hour, amount: -10, balanceBefore: 10, balanceAfter: 0},
			},
		},
		{
			name: "partially used credit lists only unused expiry",
			plans: []expiredListingPlanSetup{
				{amount: 10, release: 6, expiresAfter: 10 * time.Hour},
			},
			expectedExpired: []expectedExpiredListingTransaction{
				{expiresAfter: 10 * time.Hour, amount: -4, balanceBefore: 4, balanceAfter: 0},
			},
		},
		{
			name: "fully used credit has no expiry row",
			plans: []expiredListingPlanSetup{
				{amount: 10, release: 10, expiresAfter: 10 * time.Hour},
			},
		},
		{
			name: "reopened breakage increases listed expiry",
			plans: []expiredListingPlanSetup{
				{amount: 10, release: 6, reopen: 2, expiresAfter: 10 * time.Hour},
			},
			expectedExpired: []expectedExpiredListingTransaction{
				{expiresAfter: 10 * time.Hour, amount: -6, balanceBefore: 6, balanceAfter: 0},
			},
		},
		{
			name: "multiple expiries list independently in transaction order",
			plans: []expiredListingPlanSetup{
				{amount: 10, release: 2, expiresAfter: 10 * time.Hour},
				{amount: 7, expiresAfter: 20 * time.Hour},
			},
			expectedExpired: []expectedExpiredListingTransaction{
				{expiresAfter: 20 * time.Hour, amount: -7, balanceBefore: 7, balanceAfter: 0},
				{expiresAfter: 10 * time.Hour, amount: -8, balanceBefore: 15, balanceAfter: 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			t.Cleanup(func() {
				clock.UnFreeze()
				clock.ResetTime()
			})

			env.bookExpiredListingState(t, issuedAt, tt.plans)

			firstExpiry := issuedAt.Add(tt.plans[0].expiresAfter)
			lastExpiry := issuedAt.Add(tt.plans[len(tt.plans)-1].expiresAfter)

			expiredType := CreditTransactionTypeExpired
			beforeExpiry := firstExpiry.Add(-time.Nanosecond)
			before, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID: env.CustomerID,
				Limit:      20,
				Type:       &expiredType,
				Currency:   &env.Currency,
				AsOf:       &beforeExpiry,
			})
			require.NoError(t, err)
			require.Empty(t, before.Items)

			expired, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID: env.CustomerID,
				Limit:      20,
				Type:       &expiredType,
				Currency:   &env.Currency,
				AsOf:       &lastExpiry,
			})
			require.NoError(t, err)
			requireExpiredTransactions(t, issuedAt, expired.Items, tt.expectedExpired)
		})
	}
}

func TestListCreditTransactionsCombinesFundedConsumedAndExpired(t *testing.T) {
	issuedAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		amount   int64
		consumed int64
		asOf     time.Duration
		expected []expectedCreditTransaction
	}{
		{
			name:     "before expiry lists consumed and funded only",
			amount:   20,
			consumed: 5,
			asOf:     2 * time.Hour,
			expected: []expectedCreditTransaction{
				{txType: CreditTransactionTypeConsumed, bookedAfter: 59 * time.Minute, amount: -5, balanceBefore: 20, balanceAfter: 15},
				{txType: CreditTransactionTypeFunded, bookedAfter: 0, amount: 20, balanceBefore: 0, balanceAfter: 20},
			},
		},
		{
			name:   "unused credit at expiry lists expired and funded",
			amount: 20,
			asOf:   10 * time.Hour,
			expected: []expectedCreditTransaction{
				{txType: CreditTransactionTypeExpired, bookedAfter: 10 * time.Hour, amount: -20, balanceBefore: 20, balanceAfter: 0},
				{txType: CreditTransactionTypeFunded, bookedAfter: 0, amount: 20, balanceBefore: 0, balanceAfter: 20},
			},
		},
		{
			name:     "partially used credit at expiry lists expired consumed and funded",
			amount:   20,
			consumed: 5,
			asOf:     10 * time.Hour,
			expected: []expectedCreditTransaction{
				{txType: CreditTransactionTypeExpired, bookedAfter: 10 * time.Hour, amount: -15, balanceBefore: 15, balanceAfter: 0},
				{txType: CreditTransactionTypeConsumed, bookedAfter: 59 * time.Minute, amount: -5, balanceBefore: 20, balanceAfter: 15},
				{txType: CreditTransactionTypeFunded, bookedAfter: 0, amount: 20, balanceBefore: 0, balanceAfter: 20},
			},
		},
		{
			name:     "fully used credit at expiry lists consumed and funded only",
			amount:   20,
			consumed: 20,
			asOf:     10 * time.Hour,
			expected: []expectedCreditTransaction{
				{txType: CreditTransactionTypeConsumed, bookedAfter: 59 * time.Minute, amount: -20, balanceBefore: 20, balanceAfter: 0},
				{txType: CreditTransactionTypeFunded, bookedAfter: 0, amount: 20, balanceBefore: 0, balanceAfter: 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			t.Cleanup(func() {
				clock.UnFreeze()
				clock.ResetTime()
			})

			env.bookMixedCreditTransactionState(t, issuedAt, tt.amount, tt.consumed, 10*time.Hour)

			asOf := issuedAt.Add(tt.asOf)
			all, err := env.Service.ListCreditTransactions(t.Context(), ListCreditTransactionsInput{
				CustomerID: env.CustomerID,
				Limit:      20,
				Currency:   &env.Currency,
				AsOf:       &asOf,
			})
			require.NoError(t, err)

			requireCreditTransactions(t, issuedAt, all.Items, tt.expected)
		})
	}
}

type expiredListingPlanSetup struct {
	amount       int64
	release      int64
	reopen       int64
	expiresAfter time.Duration
}

type expectedExpiredListingTransaction struct {
	expiresAfter  time.Duration
	amount        int64
	balanceBefore int64
	balanceAfter  int64
}

type expectedCreditTransaction struct {
	txType        CreditTransactionType
	bookedAfter   time.Duration
	amount        int64
	balanceBefore int64
	balanceAfter  int64
}

func (e *testEnv) bookExpiredListingState(t *testing.T, issuedAt time.Time, specs []expiredListingPlanSetup) {
	t.Helper()

	require.True(t, slices.IsSortedFunc(specs, func(a, b expiredListingPlanSetup) int {
		return cmp.Compare(a.expiresAfter, b.expiresAfter)
	}), "test setup expects plans sorted by expiry")

	total := alpacadecimal.Zero
	for _, spec := range specs {
		total = total.Add(alpacadecimal.NewFromInt(spec.amount))
	}

	clock.FreezeTime(issuedAt)
	e.bookFBOBalance(t, total)
	e.fundOpenReceivable(t, total)

	for _, spec := range specs {
		inputs, pending, err := e.BreakageService.PlanIssuance(t.Context(), ledgerbreakage.PlanIssuanceInput{
			CustomerID: e.CustomerID,
			Amount:     alpacadecimal.NewFromInt(spec.amount),
			Currency:   e.Currency,
			ExpiresAt:  issuedAt.Add(spec.expiresAfter),
		})
		require.NoError(t, err)

		e.commitBreakageRecords(t, inputs, pending)
	}

	plans, err := e.BreakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: e.CustomerID,
		Currency:   e.Currency,
		AsOf:       issuedAt,
	})
	require.NoError(t, err)
	require.Len(t, plans, len(specs))

	for idx, spec := range specs {
		if spec.release == 0 {
			continue
		}

		plan := plans[idx]
		releaseAmount := alpacadecimal.NewFromInt(spec.release)
		usageAt := issuedAt.Add(time.Hour + time.Duration(idx)*time.Minute)

		clock.FreezeTime(usageAt)
		e.bookFBOUsage(t, usageAt, plan.FBOAddress, releaseAmount)

		releaseInput, releaseRecord, err := e.BreakageService.ReleasePlan(t.Context(), ledgerbreakage.ReleasePlanInput{
			Plan:       plan,
			Amount:     releaseAmount,
			SourceKind: ledgerbreakage.SourceKindUsage,
		})
		require.NoError(t, err)

		e.commitBreakageRecords(t, []ledger.TransactionInput{releaseInput}, []ledgerbreakage.PendingRecord{releaseRecord})

		if spec.reopen == 0 {
			continue
		}

		reopenAmount := alpacadecimal.NewFromInt(spec.reopen)
		reopenAt := usageAt.Add(time.Minute)

		clock.FreezeTime(reopenAt)
		e.bookFBORestore(t, reopenAt, reopenAmount)
		e.fundOpenReceivable(t, reopenAmount)

		reopenInput, reopenRecord, err := e.BreakageService.ReopenRelease(t.Context(), ledgerbreakage.ReopenReleaseInput{
			Release: ledgerbreakage.Release{
				Record:          releaseRecord.Record,
				OpenAmount:      releaseAmount,
				FBOAddress:      plan.FBOAddress,
				BreakageAddress: plan.BreakageAddress,
			},
			Amount:     reopenAmount,
			SourceKind: ledgerbreakage.SourceKindUsageCorrection,
		})
		require.NoError(t, err)

		e.commitBreakageRecords(t, []ledger.TransactionInput{reopenInput}, []ledgerbreakage.PendingRecord{reopenRecord})
	}
}

func (e *testEnv) bookMixedCreditTransactionState(t *testing.T, issuedAt time.Time, amount, consumed int64, expiresAfter time.Duration) {
	t.Helper()

	total := alpacadecimal.NewFromInt(amount)
	e.createPromotionalCreditFunding(t, issuedAt, total, issuedAt.Add(expiresAfter))

	if consumed == 0 {
		return
	}

	consumedAt := issuedAt.Add(time.Hour)
	clock.FreezeTime(consumedAt)

	charge := e.createFlatFeeCharge(t, alpacadecimal.NewFromInt(consumed), productcatalog.CreditOnlySettlementMode, e.sp())
	e.advanceFlatFeeCharge(t, charge)
}

func (e *testEnv) createPromotionalCreditFunding(t *testing.T, fundedAt time.Time, amount alpacadecimal.Decimal, expiresAt time.Time) creditpurchase.Charge {
	t.Helper()

	clock.FreezeTime(fundedAt)

	servicePeriod := timeutil.ClosedPeriod{
		From: fundedAt,
		To:   fundedAt.Add(time.Hour),
	}

	result, err := e.creditPurchase.Create(t.Context(), creditpurchase.CreateInput{
		Namespace: e.Namespace,
		Intent: creditpurchase.Intent{
			Intent: chargemeta.Intent{
				Name:              "Funding",
				ManagedBy:         billing.SubscriptionManagedLine,
				CustomerID:        e.CustomerID.ID,
				Currency:          e.Currency,
				ServicePeriod:     servicePeriod,
				BillingPeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
			},
			CreditAmount: amount,
			ExpiresAt:    &expiresAt,
			Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Charge.Realizations.CreditGrantRealization)

	return result.Charge
}

func (e *testEnv) commitBreakageRecords(t *testing.T, inputs []ledger.TransactionInput, pending []ledgerbreakage.PendingRecord) {
	t.Helper()

	group, err := e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
	require.NoError(t, e.BreakageService.PersistCommittedRecords(t.Context(), pending, group))
}

func (e *testEnv) bookFBOUsage(t *testing.T, at time.Time, address ledger.PostingAddress, amount alpacadecimal.Decimal) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: e.Deps.ResolversService,
			AccountCatalog: e.Deps.AccountService,
			BalanceQuerier: e.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.TransferCustomerFBOToAccruedTemplate{
			At:       at,
			Currency: e.Currency,
			Sources: []transactions.PostingAmount{
				{
					Address: address,
					Amount:  amount,
				},
			},
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *testEnv) bookFBORestore(t *testing.T, at time.Time, amount alpacadecimal.Decimal) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: e.Deps.ResolversService,
			AccountCatalog: e.Deps.AccountService,
			BalanceQuerier: e.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:       at,
			Amount:   amount,
			Currency: e.Currency,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func requireExpiredTransactions(
	t *testing.T,
	issuedAt time.Time,
	actual []CreditTransaction,
	expected []expectedExpiredListingTransaction,
) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for idx, expectedItem := range expected {
		item := actual[idx]

		require.Equal(t, CreditTransactionTypeExpired, item.Type)
		require.True(t, issuedAt.Add(expectedItem.expiresAfter).Equal(item.BookedAt), "expired booked_at at index %d", idx)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.amount).Equal(item.Amount), "expired amount at index %d: %s", idx, item.Amount)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.balanceBefore).Equal(item.Balance.Before), "expired balance before at index %d: %s", idx, item.Balance.Before)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.balanceAfter).Equal(item.Balance.After), "expired balance after at index %d: %s", idx, item.Balance.After)
	}
}

func requireCreditTransactions(
	t *testing.T,
	issuedAt time.Time,
	actual []CreditTransaction,
	expected []expectedCreditTransaction,
) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for idx, expectedItem := range expected {
		item := actual[idx]

		require.Equal(t, expectedItem.txType, item.Type, "transaction type at index %d", idx)
		require.True(t, issuedAt.Add(expectedItem.bookedAfter).Equal(item.BookedAt), "booked_at at index %d", idx)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.amount).Equal(item.Amount), "amount at index %d: %s", idx, item.Amount)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.balanceBefore).Equal(item.Balance.Before), "balance before at index %d: %s", idx, item.Balance.Before)
		require.True(t, alpacadecimal.NewFromInt(expectedItem.balanceAfter).Equal(item.Balance.After), "balance after at index %d: %s", idx, item.Balance.After)
	}
}

func requireCreditTransactionTypes(t *testing.T, actual []CreditTransaction, expected []CreditTransactionType) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for idx, expectedType := range expected {
		require.Equal(t, expectedType, actual[idx].Type, "transaction type at index %d", idx)
	}
}
