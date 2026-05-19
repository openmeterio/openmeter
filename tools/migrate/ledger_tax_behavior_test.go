package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

// TestLedgerTaxBehaviorMigrationRollback documents the rollback guard in
// 20260506103900_add_ledger_tax_behavior:
//
//   - A sub-account route created with TaxBehavior (V2 routing key) has its
//     routing_key column encoding tax_behavior:exclusive.
//   - Rolling back while that row exists would leave a V2 route pointing to a
//     column that no longer exists.
//   - The down migration fails loudly instead; callers must downgrade/remove V2
//     routes before retrying rollback.
func TestLedgerTaxBehaviorMigrationRollback(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)
	defer testDB.PGDriver.Close()

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	defer func() {
		err1, err2 := migrator.Close()
		require.NoError(t, err1)
		require.NoError(t, err2)
	}()

	require.NoError(t, migrator.Migrate(20260506103900))

	db := testDB.PGDriver.DB()
	accountID := ulid.Make().String()
	routeID := ulid.Make().String()

	const (
		namespace = "test-ledger-tax-behavior-rollback"

		// V2 routing key: currency + tax_code + tax_behavior + remaining segments.
		v2KeyVersion = "v2"
		v2KeyValue   = "currency:USD|tax_code:|tax_behavior:exclusive|features:|cost_basis:|credit_priority:|transaction_authorization_status:"
	)

	insertTaxBehaviorRoute(t, db, accountID, routeID, namespace, v2KeyVersion, v2KeyValue)

	err = migrator.Migrate(20260506102300)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot rollback: V2 routing key rows exist; downgrade routes first")

	var (
		gotVersion     string
		gotKey         string
		gotTaxBehavior string
	)
	err = db.QueryRow(`
		SELECT routing_key_version, routing_key, tax_behavior
		FROM ledger_sub_account_routes WHERE id = $1
	`, routeID).Scan(&gotVersion, &gotKey, &gotTaxBehavior)
	require.NoError(t, err)
	require.Equal(t, v2KeyVersion, gotVersion)
	require.Equal(t, v2KeyValue, gotKey)
	require.Equal(t, "exclusive", gotTaxBehavior)
}

func insertTaxBehaviorRoute(
	t *testing.T,
	db *sql.DB,
	accountID string,
	routeID string,
	namespace string,
	v2KeyVersion string,
	v2KeyValue string,
) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO ledger_accounts (id, namespace, created_at, updated_at, account_type)
		VALUES ($1, $2, NOW(), NOW(), 'customer_fbo')
	`, accountID, namespace)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO ledger_sub_account_routes (
			id, namespace, created_at, updated_at,
			routing_key_version, routing_key,
			account_id, currency, tax_behavior
		) VALUES ($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7)
	`, routeID, namespace, v2KeyVersion, v2KeyValue, accountID, "USD", "exclusive")
	require.NoError(t, err)
}
