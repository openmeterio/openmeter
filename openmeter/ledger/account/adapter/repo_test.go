package adapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestRepo_CreateAndGetAccount(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	created, err := env.repo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: namespace,
		Type:      ledger.AccountTypeCustomerFBO,
		Annotations: models.Annotations{
			"owner": "acme",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created)

	require.Equal(t, namespace, created.ID.Namespace)
	require.NotEmpty(t, created.ID.ID)
	require.Equal(t, ledger.AccountTypeCustomerFBO, created.AccountType)

	got, err := env.repo.GetAccountByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.AccountType, got.AccountType)

	entity, err := env.client.LedgerAccount.Query().
		Where(
			ledgeraccountdb.Namespace(created.ID.Namespace),
			ledgeraccountdb.ID(created.ID.ID),
		).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, models.Annotations{"owner": "acme"}, entity.Annotations)
}

func TestRepo_GetAccountByID_NotFound(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	_, err := env.repo.GetAccountByID(t.Context(), models.NamespacedID{
		Namespace: testNamespace(),
		ID:        ulid.Make().String(),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to get ledger account by id")
}

func TestRepo_CreateAndGetDimension(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	value := ulid.Make().String()

	created, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        value,
		DisplayValue: "USD",
		Annotations: models.Annotations{
			"source": "test",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, namespace, created.Namespace)
	require.NotEmpty(t, created.ID)
	require.Equal(t, ledger.DimensionKeyCurrency, created.DimensionKey)
	require.Equal(t, value, created.DimensionValue)
	require.Equal(t, "USD", created.DimensionDisplayValue)
	require.Equal(t, models.Annotations{"source": "test"}, created.Annotations)

	got, err := env.repo.GetDimensionByID(ctx, models.NamespacedID{
		Namespace: created.Namespace,
		ID:        created.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.DimensionKey, got.DimensionKey)
	require.Equal(t, created.DimensionValue, got.DimensionValue)
	require.Equal(t, created.DimensionDisplayValue, got.DimensionDisplayValue)
}

func TestRepo_CreateDimension_InvalidKey(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	_, err := env.repo.CreateDimension(t.Context(), ledgeraccount.CreateDimensionInput{
		Namespace:    testNamespace(),
		Key:          "invalid-key",
		Value:        ulid.Make().String(),
		DisplayValue: "INV",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid dimension key")
}

func TestRepo_ListSubAccounts(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	currencyA, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "USD",
		DisplayValue: "USD",
	})
	require.NoError(t, err)

	currencyB, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "EUR",
		DisplayValue: "EUR",
	})
	require.NoError(t, err)

	accountA, err := env.repo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: namespace,
		Type:      ledger.AccountTypeCustomerFBO,
	})
	require.NoError(t, err)

	accountB, err := env.repo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{
		Namespace: namespace,
		Type:      ledger.AccountTypeCustomerFBO,
	})
	require.NoError(t, err)

	subA1, err := env.repo.CreateSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Dimensions: ledgeraccount.SubAccountDimensionInput{
			CurrencyDimensionID: currencyA.ID,
		},
	})
	require.NoError(t, err)

	_, err = env.repo.CreateSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Dimensions: ledgeraccount.SubAccountDimensionInput{
			CurrencyDimensionID: currencyB.ID,
		},
	})
	require.NoError(t, err)

	_, err = env.repo.CreateSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountB.ID.ID,
		Dimensions: ledgeraccount.SubAccountDimensionInput{
			CurrencyDimensionID: currencyA.ID,
		},
	})
	require.NoError(t, err)

	t.Run("filters by namespace/account", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
		})
		require.NoError(t, err)
		require.Len(t, items, 2)
	})

	t.Run("filters by dimensions", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Dimensions: ledger.QueryDimensions{
				CurrencyID: currencyA.ID,
			},
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subA1.ID, items[0].ID)
	})
}

type TestEnv struct {
	repo   ledgeraccount.Repo
	client *entdb.Client
	db     *testutils.TestDB
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	return &TestEnv{
		repo:   adapter.NewRepo(client),
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

func testNamespace() string {
	return fmt.Sprintf("ledger-account-adapter-%d", time.Now().UnixNano())
}
