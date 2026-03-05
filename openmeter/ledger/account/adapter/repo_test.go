package adapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
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

func TestRepo_CreateDimension_AlreadyExists(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	created, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "USD",
		DisplayValue: "US Dollar",
	})
	require.NoError(t, err)
	require.NotNil(t, created)

	_, err = env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "USD",
		DisplayValue: "US Dollar",
	})
	require.Error(t, err)

	issues, mapErr := models.AsValidationIssues(err)
	require.NoError(t, mapErr)
	require.NotEmpty(t, issues)
	require.Equal(t, ledgeraccount.ErrCodeDimensionConflict, issues[0].Code())
	require.Equal(t, namespace, issues[0].Attributes()["namespace"])
	require.Equal(t, string(ledger.DimensionKeyCurrency), issues[0].Attributes()["key"])
	require.Equal(t, "USD", issues[0].Attributes()["value"])
	require.Equal(t, *created, issues[0].Attributes()["existing"])
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

	priority7, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCreditPriority),
		Value:        "7",
		DisplayValue: "Priority 7",
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

	subA3Priority7, err := env.repo.CreateSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Dimensions: ledgeraccount.SubAccountDimensionInput{
			CurrencyDimensionID:       currencyA.ID,
			CreditPriorityDimensionID: &priority7.ID,
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
		require.Len(t, items, 3)
	})

	t.Run("filters by dimensions", func(t *testing.T) {
		priority := 7
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Dimensions: ledger.QueryDimensions{
				CurrencyID:     currencyA.ID,
				CreditPriority: &priority,
			},
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subA3Priority7.ID, items[0].ID)
	})

	t.Run("create uses route uniqueness", func(t *testing.T) {
		dup, err := env.repo.CreateSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Dimensions: ledgeraccount.SubAccountDimensionInput{
				CurrencyDimensionID: currencyA.ID,
			},
		})
		require.NoError(t, err)
		require.Equal(t, subA1.ID, dup.ID)
	})
}

func TestRepo_SubAccountRouteUniquenessConstraints(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	currency, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "USD",
		DisplayValue: "USD",
	})
	require.NoError(t, err)

	priority7, err := env.repo.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCreditPriority),
		Value:        "7",
		DisplayValue: "Priority 7",
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

	createRoute := func(accountID string, creditPriorityDimensionID *string) error {
		key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, ledger.SubAccountRouteInput{
			CurrencyDimensionID:       currency.ID,
			CreditPriorityDimensionID: creditPriorityDimensionID,
		})
		require.NoError(t, err)

		_, err = env.client.LedgerSubAccountRoute.Create().
			SetNamespace(namespace).
			SetAccountID(accountID).
			SetRoutingKeyVersion(key.Version()).
			SetRoutingKey(key.Value()).
			SetCurrencyDimensionID(currency.ID).
			SetNillableCreditPriorityDimensionID(creditPriorityDimensionID).
			Save(ctx)
		return err
	}

	t.Run("rejects duplicate route for same account and key", func(t *testing.T) {
		err := createRoute(accountA.ID.ID, nil)
		require.NoError(t, err)

		err = createRoute(accountA.ID.ID, nil)
		require.Error(t, err)
		require.True(t, entdb.IsConstraintError(err))
	})

	t.Run("allows same key across different accounts", func(t *testing.T) {
		err := createRoute(accountB.ID.ID, nil)
		require.NoError(t, err)
	})

	t.Run("allows different keys within same account", func(t *testing.T) {
		err := createRoute(accountA.ID.ID, &priority7.ID)
		require.NoError(t, err)
	})

	countA, err := env.client.LedgerSubAccountRoute.Query().
		Where(
			ledgersubaccountroutedb.Namespace(namespace),
			ledgersubaccountroutedb.AccountID(accountA.ID.ID),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, countA)
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
