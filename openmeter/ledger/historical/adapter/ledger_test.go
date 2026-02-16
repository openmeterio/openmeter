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
	ledger "github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type testEntryInput struct {
	address ledger.PostingAddress
	amount  alpacadecimal.Decimal
}

func (e testEntryInput) PostingAddress() ledger.PostingAddress {
	return e.address
}

func (e testEntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

var _ ledger.EntryInput = (*testEntryInput)(nil)

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
	subAccountA := env.createSubAccount(t, namespace, "acc-a")
	subAccountB := env.createSubAccount(t, namespace, "acc-b")

	hLedger := &ledgerhistorical.Ledger{}
	txInputIntf, err := hLedger.SetUpTransactionInput(ctx, time.Now().UTC(), []ledger.EntryInput{
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
				SubAccountID: subAccountA,
				AccountType:  ledger.AccountTypeCustomerFBO,
			}),
			amount: alpacadecimal.NewFromInt(-100),
		},
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
				SubAccountID: subAccountB,
				AccountType:  ledger.AccountTypeCustomerFBO,
			}),
			amount: alpacadecimal.NewFromInt(100),
		},
	})
	require.NoError(t, err)

	txInput, ok := txInputIntf.(*ledgerhistorical.TransactionInput)
	require.True(t, ok)

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
	require.Contains(t, subAccountIDs, subAccountA)
	require.Contains(t, subAccountIDs, subAccountB)
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

type TestEnv struct {
	repo   ledgerhistorical.Repo
	client *entdb.Client
	db     *testutils.TestDB
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	return &TestEnv{
		repo:   NewRepo(client),
		client: client,
		db:     db,
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

func (e *TestEnv) createSubAccount(t *testing.T, namespace string, accountID string) string {
	t.Helper()

	account, err := e.client.LedgerAccount.Create().
		SetNamespace(namespace).
		SetID(accountID).
		SetAccountType(ledger.AccountTypeCustomerFBO).
		Save(t.Context())
	require.NoError(t, err)

	dimension, err := e.client.LedgerDimension.Create().
		SetNamespace(namespace).
		SetDimensionKey(string(ledger.DimensionKeyCurrency)).
		SetDimensionValue(fmt.Sprintf("currency-%d", time.Now().UnixNano())).
		SetDimensionDisplayValue("USD").
		Save(t.Context())
	require.NoError(t, err)

	subAccount, err := e.client.LedgerSubAccount.Create().
		SetNamespace(namespace).
		SetAccountID(account.ID).
		SetCurrencyDimensionID(dimension.ID).
		Save(t.Context())
	require.NoError(t, err)

	return subAccount.ID
}

func testNamespace() string {
	return fmt.Sprintf("ledger-historical-adapter-%d", time.Now().UnixNano())
}
