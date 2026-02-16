package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledger "github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestRepo_ListEntries_ExpandDimensions_PaginatesByEntries(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	now := time.Now().UTC()

	dimRegion := env.createDimension(t, namespace, "region", "us-east-1")
	dimPlan := env.createDimension(t, namespace, "plan", "pro")
	dimTenant := env.createDimension(t, namespace, "tenant", "acme")

	tx := env.createTransaction(t, namespace, now)

	firstEntries, err := env.repo.CreateEntries(ctx, []ledgerhistorical.CreateEntryInput{
		{
			Namespace:     namespace,
			AccountID:     "acc-first",
			AccountType:   ledger.AccountTypeCustomerFBO,
			DimensionIDs:  []string{dimRegion, dimPlan, dimTenant},
			Amount:        alpacadecimal.NewFromInt(100),
			TransactionID: tx.ID,
		},
	})
	require.NoError(t, err)
	require.Len(t, firstEntries, 1)
	firstID := firstEntries[0].ID

	time.Sleep(10 * time.Millisecond)

	secondEntries, err := env.repo.CreateEntries(ctx, []ledgerhistorical.CreateEntryInput{
		{
			Namespace:     namespace,
			AccountID:     "acc-second",
			AccountType:   ledger.AccountTypeCustomerFBO,
			DimensionIDs:  []string{dimRegion},
			Amount:        alpacadecimal.NewFromInt(50),
			TransactionID: tx.ID,
		},
	})
	require.NoError(t, err)
	require.Len(t, secondEntries, 1)
	secondID := secondEntries[0].ID

	page1, err := env.repo.ListEntries(ctx, ledgerhistorical.ListEntriesInput{
		Limit: 1,
		Expand: ledgerhistorical.EntryExpand{
			Dimensions: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, page1.Items, 1)
	require.NotNil(t, page1.NextCursor)

	require.Equal(t, firstID, page1.Items[0].ID)
	require.Len(t, page1.Items[0].DimensionsExpanded, 3)
	require.Equal(t, "pro", page1.Items[0].DimensionsExpanded["plan"].DimensionValue)
	require.Equal(t, "us-east-1", page1.Items[0].DimensionsExpanded["region"].DimensionValue)
	require.Equal(t, "acme", page1.Items[0].DimensionsExpanded["tenant"].DimensionValue)

	page2, err := env.repo.ListEntries(ctx, ledgerhistorical.ListEntriesInput{
		Cursor: page1.NextCursor,
		Limit:  1,
		Expand: ledgerhistorical.EntryExpand{
			Dimensions: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, page2.Items, 1)
	require.Equal(t, secondID, page2.Items[0].ID)
	require.Len(t, page2.Items[0].DimensionsExpanded, 1)
}

func TestRepo_CreateEntries_MapsInvalidDimensionReference(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	tx := env.createTransaction(t, namespace, time.Now().UTC())

	_, err := env.repo.CreateEntries(ctx, []ledgerhistorical.CreateEntryInput{
		{
			Namespace:     namespace,
			AccountID:     "acc-invalid-dim",
			AccountType:   ledger.AccountTypeCustomerFBO,
			DimensionIDs:  []string{"missing-dimension-id"},
			Amount:        alpacadecimal.NewFromInt(42),
			TransactionID: tx.ID,
		},
	})
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err))
}

func TestRepo_ListEntries_Filters(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	bookedAtEarly := time.Now().UTC().Add(-2 * time.Hour)
	bookedAtLate := bookedAtEarly.Add(90 * time.Minute)

	txEarly := env.createTransaction(t, namespace, bookedAtEarly)
	txLate := env.createTransaction(t, namespace, bookedAtLate)

	earlyEntries, err := env.repo.CreateEntries(ctx, []ledgerhistorical.CreateEntryInput{
		{
			Namespace:     namespace,
			AccountID:     "acc-early",
			AccountType:   ledger.AccountTypeCustomerFBO,
			Amount:        alpacadecimal.NewFromInt(10),
			TransactionID: txEarly.ID,
		},
	})
	require.NoError(t, err)
	require.Len(t, earlyEntries, 1)
	earlyID := earlyEntries[0].ID

	time.Sleep(10 * time.Millisecond)

	lateEntries, err := env.repo.CreateEntries(ctx, []ledgerhistorical.CreateEntryInput{
		{
			Namespace:     namespace,
			AccountID:     "acc-late",
			AccountType:   ledger.AccountTypeCustomerFBO,
			Amount:        alpacadecimal.NewFromInt(20),
			TransactionID: txLate.ID,
		},
	})
	require.NoError(t, err)
	require.Len(t, lateEntries, 1)
	lateID := lateEntries[0].ID

	accountFiltered, err := env.repo.ListEntries(ctx, ledgerhistorical.ListEntriesInput{
		Limit: 10,
		Filters: ledger.Filters{
			Account: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
				ID: models.NamespacedID{
					Namespace: namespace,
					ID:        "acc-early",
				},
				AccountType: ledger.AccountTypeCustomerFBO,
			}),
		},
	})
	require.NoError(t, err)
	require.Len(t, accountFiltered.Items, 1)
	require.Equal(t, earlyID, accountFiltered.Items[0].ID)

	transactionFiltered, err := env.repo.ListEntries(ctx, ledgerhistorical.ListEntriesInput{
		Limit: 10,
		Filters: ledger.Filters{
			TransactionID: &txLate.ID,
		},
	})
	require.NoError(t, err)
	require.Len(t, transactionFiltered.Items, 1)
	require.Equal(t, lateID, transactionFiltered.Items[0].ID)

	bookedAtFrom := bookedAtEarly.Add(30 * time.Minute)
	bookedAtFiltered, err := env.repo.ListEntries(ctx, ledgerhistorical.ListEntriesInput{
		Limit: 10,
		Filters: ledger.Filters{
			BookedAtPeriod: &timeutil.OpenPeriod{
				From: &bookedAtFrom,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, bookedAtFiltered.Items, 1)
	require.Equal(t, lateID, bookedAtFiltered.Items[0].ID)
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

func (e *TestEnv) createTransaction(t *testing.T, namespace string, bookedAt time.Time) ledgerhistorical.TransactionData {
	t.Helper()

	group, err := e.repo.CreateTransactionGroup(t.Context(), ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	tx, err := e.repo.CreateTransaction(t.Context(), ledgerhistorical.CreateTransactionInput{
		Namespace: namespace,
		GroupID:   group.ID,
		BookedAt:  bookedAt,
	})
	require.NoError(t, err)

	return tx
}

func (e *TestEnv) createDimension(t *testing.T, namespace, key, value string) string {
	t.Helper()

	dimension, err := e.client.LedgerDimension.Create().
		SetNamespace(namespace).
		SetDimensionKey(key).
		SetDimensionValue(value).
		Save(t.Context())
	require.NoError(t, err)

	return dimension.ID
}

func testNamespace() string {
	return fmt.Sprintf("ledger-historical-adapter-%d", time.Now().UnixNano())
}
