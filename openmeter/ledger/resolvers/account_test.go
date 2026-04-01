package resolvers_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgercustomeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercustomeraccount"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestAccountResolver_GetBusinessAccountsRequiresExplicitProvisioning(t *testing.T) {
	env := newResolverTestEnv(t)

	_, err := env.Deps.ResolversService.GetBusinessAccounts(t.Context(), env.namespace)
	require.Error(t, err)

	issues, issueErr := models.AsValidationIssues(err)
	require.NoError(t, issueErr)
	require.Len(t, issues, 1)
	assert.Equal(t, ledger.ErrBusinessAccountMissing.Code(), issues[0].Code())
}

func TestAccountResolver_EnsureBusinessAccountsIsIdempotent(t *testing.T) {
	env := newResolverTestEnv(t)

	const callers = 6

	var wg sync.WaitGroup
	errCh := make(chan error, callers)

	for i := 0; i < callers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := env.Deps.ResolversService.EnsureBusinessAccounts(context.Background(), env.namespace)
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	accounts, err := env.Deps.ResolversService.GetBusinessAccounts(t.Context(), env.namespace)
	require.NoError(t, err)
	require.NotNil(t, accounts.WashAccount)
	require.NotNil(t, accounts.EarningsAccount)
	require.NotNil(t, accounts.BrokerageAccount)

	count, err := env.DB.LedgerAccount.Query().
		Where(
			ledgeraccountdb.Namespace(env.namespace),
			ledgeraccountdb.AccountTypeIn(
				ledger.AccountTypeWash,
				ledger.AccountTypeEarnings,
				ledger.AccountTypeBrokerage,
			),
		).
		Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAccountResolver_CreateCustomerAccountsIsIdempotent(t *testing.T) {
	env := newResolverTestEnv(t)

	customerID := customer.CustomerID{
		Namespace: env.namespace,
		ID:        "customer-1",
	}

	const callers = 6

	var wg sync.WaitGroup
	errCh := make(chan error, callers)

	for i := 0; i < callers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := env.Deps.ResolversService.CreateCustomerAccounts(context.Background(), customerID)
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	accounts, err := env.Deps.ResolversService.GetCustomerAccounts(t.Context(), customerID)
	require.NoError(t, err)
	require.NotNil(t, accounts.FBOAccount)
	require.NotNil(t, accounts.ReceivableAccount)
	require.NotNil(t, accounts.AccruedAccount)

	mappingCount, err := env.DB.LedgerCustomerAccount.Query().
		Where(
			ledgercustomeraccountdb.Namespace(env.namespace),
			ledgercustomeraccountdb.CustomerID(customerID.ID),
		).
		Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, mappingCount)

	accountCount, err := env.DB.LedgerAccount.Query().
		Where(
			ledgeraccountdb.Namespace(env.namespace),
			ledgeraccountdb.AccountTypeIn(
				ledger.AccountTypeCustomerFBO,
				ledger.AccountTypeCustomerReceivable,
				ledger.AccountTypeCustomerAccrued,
			),
		).
		Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, accountCount)
}

type resolverTestEnv struct {
	DB        *entdb.Client
	Deps      ledgertestutils.Deps
	namespace string
}

func newResolverTestEnv(t *testing.T) resolverTestEnv {
	t.Helper()

	testDB := testutils.InitPostgresDB(t)
	t.Cleanup(func() {
		require.NoError(t, testDB.EntDriver.Close())
		require.NoError(t, testDB.PGDriver.Close())
	})

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		srcErr, dbErr := migrator.Close()
		require.NoError(t, srcErr)
		require.NoError(t, dbErr)
	})
	require.NoError(t, migrator.Up())

	dbClient := testDB.EntDriver.Client()
	deps, err := ledgertestutils.InitDeps(dbClient, testutils.NewDiscardLogger(t))
	require.NoError(t, err)

	return resolverTestEnv{
		DB:        dbClient,
		Deps:      deps,
		namespace: fmt.Sprintf("resolver-test-%d", time.Now().UnixNano()),
	}
}
