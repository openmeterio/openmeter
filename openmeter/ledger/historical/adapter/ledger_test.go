package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	transactionstestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestRepo_CreateTransactionGroup(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	annotations := models.Annotations{"source": "adapter-test"}

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace:   namespace,
		Annotations: annotations,
	})
	require.NoError(t, err)
	require.Equal(t, namespace, group.Namespace)
	require.Equal(t, annotations, group.Annotations)

	entity, err := env.client.LedgerTransactionGroup.Query().
		Where(
			ledgertransactiongroupdb.Namespace(namespace),
			ledgertransactiongroupdb.ID(group.ID),
		).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, annotations, entity.Annotations)
}

func TestRepo_BookTransaction_CreatesTransactionAndEntries(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	subAccountA := env.createSubAccount(t, namespace, ledger.Route{Currency: "USD"})
	subAccountB := env.createSubAccount(t, namespace, ledger.Route{Currency: "EUR"})

	txInput := mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		{
			Address:     testAddress(subAccountA),
			AmountValue: alpacadecimal.NewFromInt(-100),
		},
		{
			Address:     testAddress(subAccountB),
			AmountValue: alpacadecimal.NewFromInt(100),
		},
	})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	tx, err := env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, txInput)
	require.NoError(t, err)
	require.NotNil(t, tx)

	transactions, err := env.client.LedgerTransaction.Query().
		Where(
			ledgertransactiondb.Namespace(namespace),
			ledgertransactiondb.GroupID(group.ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, transactions, 1)

	entries, err := env.client.LedgerEntry.Query().
		Where(
			ledgerentrydb.Namespace(namespace),
			ledgerentrydb.TransactionID(transactions[0].ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	subAccountIDs := lo.Map(entries, func(e *entdb.LedgerEntry, _ int) string {
		return e.SubAccountID
	})
	require.Contains(t, subAccountIDs, subAccountA.ID)
	require.Contains(t, subAccountIDs, subAccountB.ID)

	require.Len(t, tx.Entries(), 2)
	addressesBySubAccount := map[string]ledger.PostingAddress{}
	for _, entry := range tx.Entries() {
		addr := entry.PostingAddress()
		addressesBySubAccount[addr.SubAccountID()] = addr
	}
	require.Equal(t, subAccountA.RouteMeta.RoutingKey, addressesBySubAccount[subAccountA.ID].Route().RoutingKey().Value())
	require.Equal(t, ledger.RoutingKeyVersionV1, addressesBySubAccount[subAccountA.ID].Route().RoutingKey().Version())
	require.Equal(t, subAccountB.RouteMeta.RoutingKey, addressesBySubAccount[subAccountB.ID].Route().RoutingKey().Value())
	require.Equal(t, ledger.RoutingKeyVersionV1, addressesBySubAccount[subAccountB.ID].Route().RoutingKey().Version())
}

func TestRepo_BookTransaction_NilInput(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "transaction input is required")
}

func TestRepo_ListTransactions_PaginatesAndFilters(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	subAccountA := env.createSubAccount(t, namespace, ledger.Route{Currency: "USD"})
	subAccountB := env.createSubAccount(t, namespace, ledger.Route{Currency: "EUR"})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	txInput1 := mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		{
			Address:     testAddress(subAccountA),
			AmountValue: alpacadecimal.NewFromInt(-10),
		},
		{
			Address:     testAddress(subAccountB),
			AmountValue: alpacadecimal.NewFromInt(10),
		},
	})
	tx1, err := env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInput1)
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	txInput2 := mustSetUpHistoricalTransactionInput(t, time.Now().UTC(), []*transactionstestutils.AnyEntryInput{
		{
			Address:     testAddress(subAccountA),
			AmountValue: alpacadecimal.NewFromInt(-20),
		},
		{
			Address:     testAddress(subAccountB),
			AmountValue: alpacadecimal.NewFromInt(20),
		},
	})
	tx2, err := env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInput2)
	require.NoError(t, err)

	page1, err := env.repo.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace: namespace,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Len(t, page1.Items, 1)
	require.NotNil(t, page1.NextCursor)
	require.Equal(t, tx1.ID(), page1.Items[0].ID())
	require.Len(t, page1.Items[0].Entries(), 2)

	page2, err := env.repo.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace: namespace,
		Limit:     1,
		Cursor:    page1.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, page2.Items, 1)
	require.Equal(t, tx2.ID(), page2.Items[0].ID())
	require.Len(t, page2.Items[0].Entries(), 2)

	tx2ID := tx2.ID()
	filtered, err := env.repo.ListTransactions(ctx, ledger.ListTransactionsInput{
		Namespace:     namespace,
		Limit:         10,
		TransactionID: &tx2ID,
	})
	require.NoError(t, err)
	require.Len(t, filtered.Items, 1)
	require.Equal(t, tx2.ID(), filtered.Items[0].ID())
}

func TestRepo_SumEntries_Filters(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	subAccountA := env.createSubAccount(t, namespace, ledger.Route{Currency: "USD", CreditPriority: lo.ToPtr(1)})
	subAccountB := env.createSubAccount(t, namespace, ledger.Route{Currency: "USD", CreditPriority: lo.ToPtr(2)})
	subAccountC := env.createSubAccount(t, namespace, ledger.Route{Currency: "EUR", CreditPriority: lo.ToPtr(1)})

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{Namespace: namespace})
	require.NoError(t, err)

	bookedAtEarly := time.Now().UTC().Add(-2 * time.Hour)
	txInputEarly := mustSetUpHistoricalTransactionInput(t, bookedAtEarly, []*transactionstestutils.AnyEntryInput{
		{
			Address:     testAddress(subAccountA),
			AmountValue: alpacadecimal.NewFromInt(100),
		},
		{
			Address:     testAddress(subAccountB),
			AmountValue: alpacadecimal.NewFromInt(-100),
		},
	})
	txEarly, err := env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInputEarly)
	require.NoError(t, err)

	bookedAtLate := time.Now().UTC().Add(-30 * time.Minute)
	txInputLate := mustSetUpHistoricalTransactionInput(t, bookedAtLate, []*transactionstestutils.AnyEntryInput{
		{
			Address:     testAddress(subAccountA),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
		{
			Address:     testAddress(subAccountC),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
	})
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInputLate)
	require.NoError(t, err)

	// Sum by currency
	sumUSD, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Route: ledger.RouteFilter{Currency: "USD"},
		},
	})
	require.NoError(t, err)
	// subAccountA(USD,p1): 100+50=150, subAccountB(USD,p2): -100 => total=50
	require.True(t, sumUSD.Equal(alpacadecimal.NewFromInt(50)))

	// Sum by currency + credit priority
	creditPriority := 1
	sumPriority, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Route: ledger.RouteFilter{
				Currency:       "USD",
				CreditPriority: &creditPriority,
			},
		},
	})
	require.NoError(t, err)
	// Only subAccountA(USD,p1): 100+50=150
	require.True(t, sumPriority.Equal(alpacadecimal.NewFromInt(150)))

	// Sum by transaction ID
	txID := txEarly.ID().ID
	sumTxID, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			TransactionID: &txID,
			Route:         ledger.RouteFilter{Currency: "USD"},
		},
	})
	require.NoError(t, err)
	// 100 + (-100) = 0
	require.True(t, sumTxID.Equal(alpacadecimal.NewFromInt(0)))

	// Sum by booked_at period
	from := bookedAtLate.Add(-1 * time.Minute)
	sumLate, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			BookedAtPeriod: &timeutil.OpenPeriod{From: &from},
			Route:          ledger.RouteFilter{Currency: "USD"},
		},
	})
	require.NoError(t, err)
	// Only late tx: subAccountA(USD): +50
	require.True(t, sumLate.Equal(alpacadecimal.NewFromInt(50)))
}

func TestSumEntriesQuery_SQL(t *testing.T) {
	bookedFrom := time.Now().UTC().Add(-1 * time.Hour)
	txID := "01TESTTXID1234567890123456"

	q := sumEntriesQuery{
		query: ledger.Query{
			Namespace: "ns-test",
			Filters: ledger.Filters{
				TransactionID: &txID,
				BookedAtPeriod: &timeutil.OpenPeriod{
					From: &bookedFrom,
				},
				Route: ledger.RouteFilter{
					Currency:       "USD",
					CreditPriority: lo.ToPtr(7),
				},
			},
		},
	}

	sqlStr, args := q.SQL()

	require.Equal(t, `SELECT SUM("ledger_entries"."amount") AS "sum_amount" FROM "ledger_entries" WHERE (("ledger_entries"."namespace" = $1 AND "ledger_entries"."transaction_id" = $2) AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions" WHERE "ledger_entries"."transaction_id" = "ledger_transactions"."id" AND "ledger_transactions"."booked_at" >= $3)) AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts" WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id" AND EXISTS (SELECT "ledger_sub_account_routes"."id" FROM "ledger_sub_account_routes" WHERE ("ledger_sub_accounts"."route_id" = "ledger_sub_account_routes"."id" AND "ledger_sub_account_routes"."currency" = $4) AND "ledger_sub_account_routes"."credit_priority" = $5))`, sqlStr)
	require.Equal(t, []any{
		"ns-test",
		txID,
		bookedFrom,
		"USD",
		7,
	}, args)
}

// ----------------------------------------------------------------------------
// Test helpers
// ----------------------------------------------------------------------------

type TestEnv struct {
	repo        ledgerhistorical.Repo
	accountRepo ledgeraccount.Repo
	client      *entdb.Client
	db          *testutils.TestDB
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	return &TestEnv{
		repo:        NewRepo(client),
		accountRepo: accountadapter.NewRepo(client),
		client:      client,
		db:          db,
	}
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: e.db.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	defer func() {
		srcErr, dbErr := migrator.Close()
		require.NoError(t, srcErr)
		require.NoError(t, dbErr)
	}()

	require.NoError(t, migrator.Up())
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	require.NoError(t, e.client.Close())
	require.NoError(t, e.db.EntDriver.Close())
	require.NoError(t, e.db.PGDriver.Close())
}

// createSubAccount creates an account + sub-account via the account repo for the given route.
func (e *TestEnv) createSubAccount(t *testing.T, namespace string, route ledger.Route) *ledgeraccount.SubAccountData {
	t.Helper()

	ctx := t.Context()

	acc, err := e.accountRepo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: namespace,
		Type:      ledger.AccountTypeCustomerFBO,
	})
	require.NoError(t, err)

	sub, err := e.accountRepo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: acc.ID.ID,
		Route:     route,
	})
	require.NoError(t, err)

	return sub
}

func testNamespace() string {
	return fmt.Sprintf("ledger-historical-adapter-%d", time.Now().UnixNano())
}

func mustSetUpHistoricalTransactionInput(_ *testing.T, bookedAt time.Time, entries []*transactionstestutils.AnyEntryInput) ledger.TransactionInput {
	return &transactionstestutils.AnyTransactionInput{
		BookedAtValue:     bookedAt,
		EntryInputsValues: entries,
	}
}

func testAddress(sub *ledgeraccount.SubAccountData) ledger.PostingAddress {
	return ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID:      sub.ID,
		AccountType:       sub.AccountType,
		RouteID:           sub.RouteMeta.ID,
		RoutingKeyVersion: sub.RouteMeta.RoutingKeyVersion,
		RoutingKey:        sub.RouteMeta.RoutingKey,
	})
}
