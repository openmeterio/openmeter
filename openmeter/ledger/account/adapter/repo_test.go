package adapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
		ID:        "01NONEXISTENT000000000000",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to get ledger account by id")
}

func TestRepo_ListSubAccounts(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

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

	subA1, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route:     ledger.Route{Currency: currencyx.Code("USD")},
	})
	require.NoError(t, err)

	_, err = env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route:     ledger.Route{Currency: currencyx.Code("EUR")},
	})
	require.NoError(t, err)

	subA3Priority7, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route:     ledger.Route{Currency: currencyx.Code("USD"), CreditPriority: lo.ToPtr(7)},
	})
	require.NoError(t, err)

	subA4CostBasis, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route: ledger.Route{
			Currency:  currencyx.Code("USD"),
			CostBasis: lo.ToPtr(mustDecimal(t, "0.7")),
		},
	})
	require.NoError(t, err)

	authorizedStatus := ledger.TransactionAuthorizationStatusAuthorized
	subA5AuthorizedReceivable, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route: ledger.Route{
			Currency:                       currencyx.Code("USD"),
			TransactionAuthorizationStatus: &authorizedStatus,
		},
	})
	require.NoError(t, err)

	_, err = env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountB.ID.ID,
		Route:     ledger.Route{Currency: currencyx.Code("USD")},
	})
	require.NoError(t, err)

	t.Run("filters by namespace/account", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
		})
		require.NoError(t, err)
		require.Len(t, items, 5)
	})

	t.Run("filters by route", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route: ledger.RouteFilter{
				Currency:       currencyx.Code("USD"),
				CreditPriority: lo.ToPtr(7),
			},
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subA3Priority7.ID, items[0].ID)
	})

	t.Run("filters by canonicalized cost basis", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route: ledger.RouteFilter{
				Currency:  currencyx.Code("USD"),
				CostBasis: mo.Some(lo.ToPtr(mustDecimal(t, "0.70"))),
			},
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subA4CostBasis.ID, items[0].ID)
		require.NotNil(t, items[0].Route.CostBasis)
		require.True(t, items[0].Route.CostBasis.Equal(mustDecimal(t, "0.7")))
	})

	t.Run("filters by transaction authorization status", func(t *testing.T) {
		items, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route: ledger.RouteFilter{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &authorizedStatus,
			},
		})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, subA5AuthorizedReceivable.ID, items[0].ID)
		require.NotNil(t, items[0].Route.TransactionAuthorizationStatus)
		require.Equal(t, authorizedStatus, *items[0].Route.TransactionAuthorizationStatus)
	})

	t.Run("create uses route uniqueness", func(t *testing.T) {
		dup, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route:     ledger.Route{Currency: currencyx.Code("USD")},
		})
		require.NoError(t, err)
		require.Equal(t, subA1.ID, dup.ID)
	})

	t.Run("create canonicalizes cost basis uniqueness", func(t *testing.T) {
		dup, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route: ledger.Route{
				Currency:  currencyx.Code("USD"),
				CostBasis: lo.ToPtr(mustDecimal(t, "0.70")),
			},
		})
		require.NoError(t, err)
		require.Equal(t, subA4CostBasis.ID, dup.ID)
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

	createRoute := func(accountID string, creditPriority *int, costBasis *alpacadecimal.Decimal) error {
		key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, ledger.Route{
			Currency:       currencyx.Code("USD"),
			CostBasis:      costBasis,
			CreditPriority: creditPriority,
		})
		require.NoError(t, err)

		create := env.client.LedgerSubAccountRoute.Create().
			SetNamespace(namespace).
			SetAccountID(accountID).
			SetRoutingKeyVersion(key.Version()).
			SetRoutingKey(key.Value()).
			SetCurrency("USD").
			SetNillableCostBasis(costBasis).
			SetNillableCreditPriority(creditPriority)

		_, err = create.Save(ctx)
		return err
	}

	t.Run("rejects duplicate route for same account and key", func(t *testing.T) {
		err := createRoute(accountA.ID.ID, nil, nil)
		require.NoError(t, err)

		err = createRoute(accountA.ID.ID, nil, nil)
		require.Error(t, err)
		require.True(t, entdb.IsConstraintError(err))
	})

	t.Run("allows same key across different accounts", func(t *testing.T) {
		err := createRoute(accountB.ID.ID, nil, nil)
		require.NoError(t, err)
	})

	t.Run("allows different keys within same account", func(t *testing.T) {
		err := createRoute(accountA.ID.ID, lo.ToPtr(7), nil)
		require.NoError(t, err)
	})

	t.Run("canonical cost basis produces duplicate key", func(t *testing.T) {
		err := createRoute(accountA.ID.ID, nil, lo.ToPtr(mustDecimal(t, "0.7")))
		require.NoError(t, err)

		err = createRoute(accountA.ID.ID, nil, lo.ToPtr(mustDecimal(t, "0.70")))
		require.Error(t, err)
		require.True(t, entdb.IsConstraintError(err))
	})

	countA, err := env.client.LedgerSubAccountRoute.Query().
		Where(
			ledgersubaccountroutedb.Namespace(namespace),
			ledgersubaccountroutedb.AccountID(accountA.ID.ID),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, countA)
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
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
