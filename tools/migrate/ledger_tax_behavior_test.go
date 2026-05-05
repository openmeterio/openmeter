package migrate_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestLedgerTaxBehaviorMigrationRollback documents the known data-loss scenario
// when rolling back 20260506103900_add_ledger_tax_behavior:
//
//   - A sub-account route created with TaxBehavior (V2 routing key) has its
//     routing_key column encoding tax_behavior:exclusive.
//   - After rollback, the tax_behavior column is dropped but the row persists.
//   - The stored routing_key still encodes the V2 format, making the row
//     unresolvable by the application until the migration is re-applied.
func TestLedgerTaxBehaviorMigrationRollback(t *testing.T) {
	accountID := ulid.Make().String()
	routeID := ulid.Make().String()

	const (
		namespace = "test-ledger-tax-behavior-rollback"

		// V2 routing key: currency + tax_code + tax_behavior + remaining segments.
		v2KeyVersion = "v2"
		v2KeyValue   = "currency:USD|tax_code:|tax_behavior:exclusive|features:|cost_basis:|credit_priority:|transaction_authorization_status:"
	)

	runner{
		stops: stops{
			{
				// After 20260506103900 is applied: tax_behavior column exists.
				// Insert a V2-keyed route row to simulate a live sub-account.
				version:   20260506103900,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
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
				},
			},
			{
				// After rolling back to 20260506102300 (one step before 20260506103900):
				// tax_behavior column is dropped. The V2 route row is now orphaned:
				// routing_key still encodes "tax_behavior:exclusive" but the column is gone.
				// The row cannot be fully reconstructed until the migration is re-applied.
				version:   20260506102300,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Row must survive the rollback — the down migration only drops the column.
					var gotID string
					err := db.QueryRow(`SELECT id FROM ledger_sub_account_routes WHERE id = $1`, routeID).Scan(&gotID)
					require.NoError(t, err, "orphaned V2 route row must survive down migration")
					require.Equal(t, routeID, gotID)

					// The routing key still encodes the V2 format with tax_behavior segment.
					var gotVersion, gotKey string
					err = db.QueryRow(`
						SELECT routing_key_version, routing_key
						FROM ledger_sub_account_routes WHERE id = $1
					`, routeID).Scan(&gotVersion, &gotKey)
					require.NoError(t, err)
					require.Equal(t, v2KeyVersion, gotVersion,
						"routing_key_version must remain v2 — orphaned row cannot self-downgrade")
					require.True(t, strings.Contains(gotKey, "tax_behavior:exclusive"),
						"routing_key must still encode tax_behavior segment — column gone but key string persists")

					// tax_behavior column must no longer exist.
					_, err = db.Exec(`SELECT tax_behavior FROM ledger_sub_account_routes LIMIT 1`)
					require.Error(t, err, "tax_behavior column must not exist after rollback")
					require.Contains(t, err.Error(), "tax_behavior",
						"postgres error must mention the missing column name")
				},
			},
		},
	}.Test(t)
}
