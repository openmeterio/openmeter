package adapter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRepo_CreateAndGetAccount(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

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
		Route:     ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))},
	})
	require.NoError(t, err)

	_, err = env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route:     ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("EUR"))},
	})
	require.NoError(t, err)

	subA3Priority7, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route:     ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD")), CreditPriority: lo.ToPtr(7)},
	})
	require.NoError(t, err)

	subA4CostBasis, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route: ledger.Route{
			Currency:  currencies.NewCurrencyReference(currencyx.Code("USD")),
			CostBasis: lo.ToPtr(mustDecimal(t, "0.7")),
		},
	})
	require.NoError(t, err)

	authorizedStatus := ledger.TransactionAuthorizationStatusAuthorized
	subA5AuthorizedReceivable, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountA.ID.ID,
		Route: ledger.Route{
			Currency:                       currencies.NewCurrencyReference(currencyx.Code("USD")),
			TransactionAuthorizationStatus: &authorizedStatus,
		},
	})
	require.NoError(t, err)

	_, err = env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountB.ID.ID,
		Route:     ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))},
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
				Currency:       currencies.NewCurrencyReference("USD"),
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
				Currency:  currencies.NewCurrencyReference("USD"),
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
				Currency:                       currencies.NewCurrencyReference("USD"),
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
			Route:     ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))},
		})
		require.NoError(t, err)
		require.Equal(t, subA1.ID, dup.ID)
	})

	t.Run("create canonicalizes cost basis uniqueness", func(t *testing.T) {
		dup, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace,
			AccountID: accountA.ID.ID,
			Route: ledger.Route{
				Currency:  currencies.NewCurrencyReference(currencyx.Code("USD")),
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
		key, err := ledger.BuildRoutingKey(ledger.Route{
			Currency:       currencies.NewCurrencyReference(currencyx.Code("USD")),
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

	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	client := db.EntDriver.Client()

	return &TestEnv{
		repo:   adapter.NewRepo(client),
		client: client,
		db:     db,
	}
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

func TestRepoFilterStorageCutover(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	ctx := t.Context()
	ns := testNamespace()
	account, err := env.repo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{Namespace: ns, Type: ledger.AccountTypeCustomerFBO})
	require.NoError(t, err)
	// Given a route created by a writer that populates the JSON representation.
	input := ledgeraccount.CreateSubAccountInput{Namespace: ns, AccountID: account.ID.ID, Route: ledger.Route{
		Currency: currencies.NewCurrencyReference(currencyx.Code("USD")), Filters: crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: []string{"output", "input"}},
	}}
	sub, err := env.repo.EnsureSubAccount(ctx, input)
	require.NoError(t, err)
	stored, err := env.client.LedgerSubAccountRoute.Query().Where(ledgersubaccountroutedb.ID(sub.RouteMeta.ID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, stored.Filters)
	require.Equal(t, crediteligibility.FiltersVersion1, stored.Filters.Version)
	require.Equal(t, []string{"input", "output"}, stored.Filters.Features)
	require.Empty(t, stored.Features)
	// When the unused legacy column is stale, lookup still preserves identity and filters.
	_, err = env.db.PGDriver.DB().ExecContext(ctx, `UPDATE ledger_sub_account_routes SET features = ARRAY['stale'] WHERE id = $1`, stored.ID)
	require.NoError(t, err)
	existing, err := env.repo.EnsureSubAccount(ctx, input)
	require.NoError(t, err)
	require.Equal(t, sub.ID, existing.ID)
	require.Equal(t, []string{"input", "output"}, existing.Route.Filters.Features)
}

func TestRepoExactFiltersSeparateFeatureAndPlanRoutes(t *testing.T) {
	// given: feature-only and plan-restricted routes share a feature.
	env := NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	ctx := t.Context()
	namespace := testNamespace()
	account, err := env.repo.CreateAccount(ctx, ledgeraccount.CreateAccountInput{Namespace: namespace, Type: ledger.AccountTypeCustomerFBO})
	require.NoError(t, err)
	featureRoute := ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD")), Filters: crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: []string{"api-calls"}}}
	feature, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{Namespace: namespace, AccountID: account.ID.ID, Route: featureRoute})
	require.NoError(t, err)
	planRoute := featureRoute
	planRoute.Filters = crediteligibility.Filters{Version: crediteligibility.FiltersVersion2, Features: []string{"api-calls"}, Plans: []crediteligibility.PlanFilter{{Key: "pro"}}}
	plan, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{Namespace: namespace, AccountID: account.ID.ID, Route: planRoute})
	require.NoError(t, err)

	// when: the same feature dimensions are stored in either supported format,
	// exact lookup must still select the feature bucket and exclude the plan bucket.
	for _, version := range []int{1, 2} {
		_, err = env.db.PGDriver.DB().ExecContext(ctx, `UPDATE ledger_sub_account_routes SET filters = jsonb_set(filters, '{schema_version}', to_jsonb($2::int)) WHERE id = $1`, feature.RouteMeta.ID, version)
		require.NoError(t, err)
		for _, tc := range []struct {
			route ledger.Route
			id    string
		}{{featureRoute, feature.ID}, {planRoute, plan.ID}} {
			found, err := env.repo.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{Namespace: namespace, AccountID: account.ID.ID, Route: tc.route.Filter()})
			require.NoError(t, err)
			require.Len(t, found, 1)
			require.Equal(t, tc.id, found[0].ID)
		}
	}
	// then: resolving the feature-only route reuses its original bucket.
	again, err := env.repo.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{Namespace: namespace, AccountID: account.ID.ID, Route: featureRoute})
	require.NoError(t, err)
	require.Equal(t, feature.ID, again.ID)
	require.Equal(t, crediteligibility.FiltersVersion2, again.Route.Filters.Version)
}
