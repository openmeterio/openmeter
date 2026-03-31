package testutils

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type IntegrationEnv struct {
	Namespace        string
	CustomerID       customer.CustomerID
	Currency         currencyx.Code
	DB               *entdb.Client
	CustomerAccounts ledger.CustomerAccounts
	BusinessAccounts ledger.BusinessAccounts
	Deps             Deps
}

type lazyQuerier struct {
	querier ledger.Querier
}

func (l *lazyQuerier) SumEntries(ctx context.Context, query ledger.Query) (ledger.QuerySummedResult, error) {
	return l.querier.SumEntries(ctx, query)
}

func NewIntegrationEnv(t *testing.T, namespacePrefix string) *IntegrationEnv {
	t.Helper()

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	t.Cleanup(clock.UnFreeze)

	testDB := omtestutils.InitPostgresDB(t)
	t.Cleanup(func() {
		require.NoError(t, testDB.EntDriver.Close())
		require.NoError(t, testDB.PGDriver.Close())
	})

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           omtestutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		srcErr, dbErr := migrator.Close()
		require.NoError(t, srcErr)
		require.NoError(t, dbErr)
	})
	require.NoError(t, migrator.Up())

	namespace := fmt.Sprintf("%s-%d", namespacePrefix, clock.Now().UnixNano())
	lq := &lazyQuerier{}

	locker, err := common.NewLocker(slog.Default())
	require.NoError(t, err)

	db := testDB.EntDriver.Client()
	accountRepo := common.NewLedgerAccountRepo(db)
	live := ledgeraccount.AccountLiveServices{
		Locker:  locker,
		Querier: lq,
	}
	accountSvc := accountservice.New(accountRepo, live)

	historicalRepo := common.NewLedgerHistoricalRepo(db)
	historicalLedger := historical.NewLedger(historicalRepo, accountSvc, locker, routingrules.DefaultValidator)
	lq.querier = historicalLedger

	resolversRepo := common.NewLedgerResolversRepo(db)
	resolversService := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountSvc,
		Repo:           resolversRepo,
	})

	deps := Deps{
		AccountService:   accountSvc,
		ResolversService: resolversService,
		HistoricalLedger: historicalLedger,
	}

	customerID := customer.CustomerID{
		Namespace: namespace,
		ID:        "customer-1",
	}
	customerAccounts, err := deps.ResolversService.CreateCustomerAccounts(t.Context(), customerID)
	require.NoError(t, err)

	businessAccounts, err := deps.ResolversService.GetBusinessAccounts(t.Context(), namespace)
	require.NoError(t, err)

	return &IntegrationEnv{
		Namespace:        namespace,
		CustomerID:       customerID,
		Currency:         currencyx.Code("USD"),
		DB:               db,
		CustomerAccounts: customerAccounts,
		BusinessAccounts: businessAccounts,
		Deps:             deps,
	}
}

func (e *IntegrationEnv) Now() time.Time {
	return clock.Now().UTC()
}

func (e *IntegrationEnv) FBOSubAccount(t *testing.T, priority int) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.Currency,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) ReceivableSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, nil, ledger.TransactionAuthorizationStatusOpen)
}

func (e *IntegrationEnv) ReceivableSubAccountWithStatus(t *testing.T, status ledger.TransactionAuthorizationStatus) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, nil, status)
}

func (e *IntegrationEnv) ReceivableSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, costBasis, ledger.TransactionAuthorizationStatusOpen)
}

func (e *IntegrationEnv) ReceivableSubAccountWithCostBasisAndStatus(t *testing.T, costBasis *alpacadecimal.Decimal, status ledger.TransactionAuthorizationStatus) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       e.Currency,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: status,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) AccruedSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.AccruedSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) AccruedSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  e.Currency,
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) WashSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.WashSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) WashSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.Currency,
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) EarningsSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.EarningsSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) EarningsSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.EarningsAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.Currency,
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) BrokerageSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.BrokerageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency: e.Currency,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) SumBalance(t *testing.T, subAccount ledger.SubAccount) alpacadecimal.Decimal {
	t.Helper()

	sum, err := subAccount.GetBalance(t.Context())
	require.NoError(t, err)

	return sum.Settled()
}
