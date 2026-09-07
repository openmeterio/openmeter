package adapter

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	transactionstestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRepoGetBalancesAtBoundariesPreservesScopeAndCursor(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	ctx := t.Context()
	namespace := testNamespace()
	alpha := mustCustomCurrencyReference(t, "ACME", "custom-currency-alpha", 2)
	beta := mustCustomCurrencyReference(t, "ACME", "custom-currency-beta", 2)
	account, err := env.accountRepo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{Namespace: namespace, Type: ledger.AccountTypeCustomerFBO})
	require.NoError(t, err)
	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{Namespace: namespace})
	require.NoError(t, err)
	bookedAt := time.Now().UTC().Truncate(time.Microsecond)
	var firstCursor ledger.TransactionCursor

	// given: reused currency codes, feature-restricted credit, and two postings
	// at the same booked time share an account but have separate balance scopes.
	for idx, posting := range []struct {
		currency currencies.CurrencyReference
		features []string
		amount   int64
	}{
		{currency: alpha, amount: 100},
		{currency: alpha, amount: -30},
		{currency: alpha, features: []string{"api"}, amount: 50},
		{currency: beta, amount: 200},
	} {
		sub, err := env.accountRepo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace, AccountID: account.ID.ID,
			Route: ledger.Route{Currency: posting.currency, Features: posting.features, CreditPriority: lo.ToPtr(1)},
		})
		require.NoError(t, err)
		counterpart := env.createSubAccountOfType(t, namespace, ledger.AccountTypeCustomerReceivable, ledger.Route{Currency: posting.currency})
		tx, err := env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID},
			mustSetUpHistoricalTransactionInput(t, bookedAt, []*transactionstestutils.AnyEntryInput{
				provenanceEntryInput(t, sub, alpacadecimal.NewFromInt(posting.amount), nil, nil),
				provenanceEntryInput(t, counterpart, alpacadecimal.NewFromInt(-posting.amount), nil, nil),
			}))
		require.NoError(t, err)
		if idx == 0 {
			firstCursor = tx.Cursor()
		}
	}

	// when: one batch mixes cursor and timestamp boundaries with exact identity
	// and feature scopes, including a scope with no matching entries.
	input := ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, After: &firstCursor, Route: ledger.RouteFilter{Currency: alpha}}},
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, AsOf: &bookedAt, Route: ledger.RouteFilter{Currency: alpha}}},
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, AsOf: &bookedAt, Route: ledger.RouteFilter{Currency: alpha, Features: mo.Some([]string{})}}},
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, AsOf: &bookedAt, Route: ledger.RouteFilter{Currency: alpha, Features: mo.Some([]string{"api"})}}},
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, AsOf: &bookedAt, Route: ledger.RouteFilter{Currency: beta}}},
		{Namespace: namespace, Filters: ledger.Filters{AccountID: &account.ID.ID, AsOf: lo.ToPtr(bookedAt.Add(-time.Second)), Route: ledger.RouteFilter{Currency: alpha}}},
	}}
	balances, err := env.repo.GetBalancesAtBoundaries(ctx, input)
	require.NoError(t, err)

	// then: results retain input ordering and include omitted movements exactly once.
	require.Len(t, balances, 6)
	for idx, expected := range []float64{100, 120, 70, 50, 200, 0} {
		require.Equal(t, expected, balances[idx].InexactFloat64(), "boundary %d", idx)
	}
}
