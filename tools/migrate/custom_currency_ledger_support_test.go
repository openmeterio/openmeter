package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	customCurrencyLedgerSupportMigration = 20260710144109
	previousMigration                    = 20260709134422
)

func TestCustomCurrencyLedgerSupportMigrationRejectsLongCurrencyRollback(t *testing.T) {
	tables := []string{
		"charge_credit_purchases",
		"charge_flat_fee_run_detailed_lines",
		"charge_flat_fees",
		"charge_usage_based",
		"charge_usage_based_run_detailed_line",
		"credit_realization_lineages",
		"ledger_breakage_records",
	}

	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			migrator, db := newCustomCurrencyMigrationTest(t)
			require.NoError(t, migrator.Migrate(customCurrencyLedgerSupportMigration))
			insertCurrencyOnlyRow(t, db, table, "ACME_CREDITS")

			err := migrator.Migrate(previousMigration)
			require.ErrorContains(t, err, "cannot rollback custom currency ledger support: currency values longer than 3 characters exist")

			var currency string
			err = db.QueryRow(fmt.Sprintf("SELECT currency FROM %s", pq.QuoteIdentifier(table))).Scan(&currency)
			require.NoError(t, err)
			require.Equal(t, "ACME_CREDITS", currency)
			requireSourceColumn(t, db, true)
			requireChargesSearchView(t, db)
		})
	}
}

func TestCustomCurrencyLedgerSupportMigrationRejectsSourceRouteRollback(t *testing.T) {
	migrator, db := newCustomCurrencyMigrationTest(t)
	require.NoError(t, migrator.Migrate(customCurrencyLedgerSupportMigration))

	accountID := ulid.Make().String()
	routeID := ulid.Make().String()
	_, err := db.Exec(`
		INSERT INTO ledger_accounts (id, namespace, created_at, updated_at, account_type)
		VALUES ($1, 'custom-currency-rollback', NOW(), NOW(), 'customer_fbo')
	`, accountID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO ledger_sub_account_routes (
			id, namespace, created_at, updated_at, routing_key_version, routing_key,
			account_id, currency, source
		) VALUES ($1, 'custom-currency-rollback', NOW(), NOW(), 'v3', $2, $3, 'ACME', 'USD')
	`, routeID, "currency:ACME|source:USD", accountID)
	require.NoError(t, err)

	err = migrator.Migrate(previousMigration)
	require.ErrorContains(t, err, "cannot rollback custom currency ledger support: source-bearing or V3 ledger routes exist")

	var (
		source  string
		version string
	)
	err = db.QueryRow(`
		SELECT source, routing_key_version
		FROM ledger_sub_account_routes
		WHERE id = $1
	`, routeID).Scan(&source, &version)
	require.NoError(t, err)
	require.Equal(t, "USD", source)
	require.Equal(t, "v3", version)
	requireChargesSearchView(t, db)
}

func TestCustomCurrencyLedgerSupportMigrationSafeUpDownUp(t *testing.T) {
	migrator, db := newCustomCurrencyMigrationTest(t)

	require.NoError(t, migrator.Migrate(customCurrencyLedgerSupportMigration))
	requireSourceColumn(t, db, true)
	requireChargesSearchView(t, db)

	require.NoError(t, migrator.Migrate(previousMigration))
	requireSourceColumn(t, db, false)
	requireChargesSearchView(t, db)

	require.NoError(t, migrator.Migrate(customCurrencyLedgerSupportMigration))
	requireSourceColumn(t, db, true)
	requireChargesSearchView(t, db)
}

func newCustomCurrencyMigrationTest(t *testing.T) (*migrate.Migrate, *sql.DB) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t)
	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		sourceErr, databaseErr := migrator.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, databaseErr)
		require.NoError(t, testDB.PGDriver.Close())
	})

	return migrator, testDB.PGDriver.DB()
}

func insertCurrencyOnlyRow(t *testing.T, db *sql.DB, table string, currency string) {
	t.Helper()

	rows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		  AND is_nullable = 'NO'
		  AND column_default IS NULL
		  AND column_name NOT IN ('id', 'currency')
	`, table)
	require.NoError(t, err)

	columns := make([]string, 0)
	for rows.Next() {
		var column string
		require.NoError(t, rows.Scan(&column))
		columns = append(columns, column)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	for _, column := range columns {
		_, err = db.Exec(fmt.Sprintf(
			"ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
			pq.QuoteIdentifier(table),
			pq.QuoteIdentifier(column),
		))
		require.NoError(t, err)
	}

	_, err = db.Exec(fmt.Sprintf(
		"INSERT INTO %s (id, currency) VALUES ($1, $2)",
		pq.QuoteIdentifier(table),
	), ulid.Make().String(), currency)
	require.NoError(t, err)
}

func requireSourceColumn(t *testing.T, db *sql.DB, expected bool) {
	t.Helper()

	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'ledger_sub_account_routes'
			  AND column_name = 'source'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.Equal(t, expected, exists)
}

func requireChargesSearchView(t *testing.T, db *sql.DB) {
	t.Helper()

	var exists bool
	err := db.QueryRow(`SELECT to_regclass('public.charges_search_v1s') IS NOT NULL`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists)
}
