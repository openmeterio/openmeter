package common

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestLedgerNamespaceHandlerIsSafeForRepeatedCreateNamespaceCalls(t *testing.T) {
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

	deps, err := ledgertestutils.InitDeps(testDB.EntDriver.Client(), testutils.NewDiscardLogger(t))
	require.NoError(t, err)

	handler := NewLedgerNamespaceHandler(deps.ResolversService)
	namespace := fmt.Sprintf("startup-test-%d", time.Now().UnixNano())

	require.NoError(t, handler.CreateNamespace(t.Context(), namespace))
	require.NoError(t, handler.CreateNamespace(t.Context(), namespace))

	count, err := testDB.EntDriver.Client().LedgerAccount.Query().
		Where(
			ledgeraccountdb.Namespace(namespace),
			ledgeraccountdb.AccountTypeIn(
				ledger.AccountTypeWash,
				ledger.AccountTypeEarnings,
				ledger.AccountTypeBrokerage,
			),
		).
		Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 3, count)
}
