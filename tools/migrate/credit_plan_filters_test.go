package migrate_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestCreditPlanFiltersRollbackPreservesAttribution(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()
	conn, err := testDB.PGDriver.DB().Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	// Given recorded attribution and a filter that feature-only readers cannot decode.
	_, err = conn.ExecContext(t.Context(), `CREATE TEMP TABLE charge_flat_fees (subscription_plan jsonb);
 INSERT INTO charge_flat_fees VALUES ('{"key":"pro","version":2}');
 CREATE TEMP TABLE charge_credit_purchases (filters jsonb);
 CREATE TEMP TABLE ledger_sub_account_routes (filters jsonb);`)
	require.NoError(t, err)
	down := readMigration(t, "20260922140147_credit_plan_filters.down.sql")
	for _, table := range []string{"charge_credit_purchases", "ledger_sub_account_routes"} {
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`INSERT INTO %s VALUES ('{"schema_version":2,"plans":[{"key":"pro"}]}')`, table))
		require.NoError(t, err)

		// When rollback is attempted, it rejects the filters before deleting attribution.
		_, err = conn.ExecContext(t.Context(), down)
		require.ErrorContains(t, err, "cannot remove plan attribution")
		_, err = conn.ExecContext(t.Context(), "ROLLBACK")
		require.NoError(t, err)

		// Then the charge snapshot is intact.
		var snapshot string
		err = conn.QueryRowContext(t.Context(), `SELECT subscription_plan::text FROM charge_flat_fees`).Scan(&snapshot)
		require.NoError(t, err)
		require.JSONEq(t, `{"key":"pro","version":2}`, snapshot)
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`DELETE FROM %s`, table))
		require.NoError(t, err)
	}
}

func TestCreditPlanFilterMigrationPreservesView(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	defer testDB.PGDriver.Close()
	conn, err := testDB.PGDriver.DB().Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	// given: a dependent view relies on the appended snapshot column.
	_, err = conn.ExecContext(t.Context(), `CREATE TEMP VIEW credit_plan_snapshot_dependency AS SELECT subscription_plan FROM charges_search_v1s`)
	require.NoError(t, err)
	var originalOID int64
	require.NoError(t, conn.QueryRowContext(t.Context(), `SELECT 'charges_search_v1s'::regclass::oid`).Scan(&originalOID))

	// when: storage is rolled back and reapplied, retaining the view's public shape.
	for _, direction := range []string{"down", "up"} {
		_, err = conn.ExecContext(t.Context(), readMigration(t, "20260922140147_credit_plan_filters."+direction+".sql"))
		require.NoError(t, err, direction)

		// then: view identity and its dependents survive both transitions.
		var currentOID int64
		require.NoError(t, conn.QueryRowContext(t.Context(), `SELECT 'charges_search_v1s'::regclass::oid`).Scan(&currentOID))
		require.Equal(t, originalOID, currentOID)
		var count int
		require.NoError(t, conn.QueryRowContext(t.Context(), `SELECT count(*) FROM credit_plan_snapshot_dependency`).Scan(&count))
	}
}
