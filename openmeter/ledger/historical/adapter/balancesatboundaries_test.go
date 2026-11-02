package adapter

import (
	"database/sql"
	"fmt"
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
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
			Route: ledger.Route{Currency: posting.currency, Filters: ledger.CreditFilters{Version: ledger.CreditFiltersVersion1, Features: posting.features}, CreditPriority: lo.ToPtr(1)},
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

func TestRepoGetBalancesAtBoundariesMatchesIndependentSums(t *testing.T) {
	db, queries := seedBalanceBoundaryHistory(t, 1000)
	ctx := t.Context()
	last := *queries[0].Filters.After
	fboID := *queries[0].Filters.AccountID

	// given: the last two transactions tie on both timestamps. The last has
	// multiple FBO entries, which must all be counted at its cursor.
	_, err := db.ExecContext(ctx, "UPDATE ledger_transactions SET booked_at = $1 WHERE id = $2", last.BookedAt, fmt.Sprintf("%026d", 999))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE ledger_entries SET amount = -0.25
		WHERE transaction_id = $1 AND sub_account_id IN (SELECT id FROM ledger_sub_accounts WHERE account_id = $2)`, last.ID.ID, fboID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ledger_entries
		(id, namespace, created_at, updated_at, amount, sub_account_id, transaction_id, identity_key, schema_version)
		SELECT '3' || substr(id, 2), namespace, created_at, updated_at, -0.75, sub_account_id, transaction_id, 'split', 1
		FROM ledger_entries WHERE transaction_id = $1 AND sub_account_id IN (SELECT id FROM ledger_sub_accounts WHERE account_id = $2)`, last.ID.ID, fboID)
	require.NoError(t, err)

	preceding := last
	preceding.ID.ID = fmt.Sprintf("%026d", 999)
	beforeTied := last
	beforeTied.CreatedAt = last.CreatedAt.Add(-time.Microsecond)
	afterTied := last
	afterTied.CreatedAt = last.CreatedAt.Add(time.Microsecond)
	at := last.BookedAt
	beforeAll := last.CreatedAt.Add(-time.Hour)
	fbo := queries[0]
	fbo.Filters.After = nil

	// when: boundaries are out of order, duplicated, or don't name an existing
	// transaction. Timestamp boundaries still include the whole booked time.
	input := ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		queries[0], queries[1], queries[0],
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, After: &preceding}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, After: &beforeTied}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, After: &afterTied}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &at}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &beforeAll}},
		{Namespace: "empty-namespace", Filters: ledger.Filters{AsOf: &at}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &at, TransactionID: &last.ID.ID}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &at, BookedAtPeriod: &timeutil.OpenPeriod{From: &at}}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &at, BookedAtPeriod: &timeutil.OpenPeriod{To: &at}}},
		{Namespace: fbo.Namespace, Filters: ledger.Filters{AccountID: &fboID, AsOf: &at, Provenance: ledger.ProvenanceFilter{SourceChargeID: mo.Some(lo.ToPtr("missing-source"))}}},
	}}
	query, args, err := (balancesAtBoundariesQuery{input: input}).SQL()
	require.NoError(t, err)
	rows, err := db.QueryContext(ctx, query, args...)
	require.NoError(t, err)
	defer rows.Close()

	// then: every requested boundary appears once and agrees with a separate
	// ledger SUM, including the transaction split and all unchanged filters.
	expected := []float64{100, -100, 100, 101, 102, 100, 100, 0, 0, -1, -2, 102, 0}
	seen := make(map[int]bool)
	for rows.Next() {
		var amount string
		var idx int
		require.NoError(t, rows.Scan(&amount, &idx))
		require.False(t, seen[idx], "duplicate boundary %d", idx)
		seen[idx] = true
		balance, err := alpacadecimal.NewFromString(amount)
		require.NoError(t, err)
		require.Equal(t, expected[idx], balance.InexactFloat64(), "boundary %d", idx)

		independentSQL, independentArgs, err := (&sumEntriesQuery{query: input.Queries[idx]}).SQL()
		require.NoError(t, err)
		var independentAmount sql.NullString
		require.NoError(t, db.QueryRowContext(ctx, independentSQL, independentArgs...).Scan(&independentAmount))
		independent, err := decimalFromNullString(independentAmount)
		require.NoError(t, err)
		require.Equal(t, independent.InexactFloat64(), balance.InexactFloat64(), "boundary %d", idx)
	}

	require.NoError(t, rows.Err())
	require.Len(t, seen, len(input.Queries))
}
