package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLedgerCustomCurrencyMigrationBackwardCompatibility(t *testing.T) {
	const (
		versionBefore = 20260724082957
		versionAfter  = 20260728160000
		namespace     = "tenant-a"
	)

	legacyAccountID := ulid.Make().String()
	legacyRouteID := ulid.Make().String()
	oldWriterRouteID := ulid.Make().String()
	legacyGroupID := ulid.Make().String()
	oldWriterGroupID := ulid.Make().String()

	runner{
		stops: stops{
			{
				version:   versionBefore,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// given:
					// - rows written by the previous application schema
					_, err := db.Exec(`
						INSERT INTO ledger_accounts (id, namespace, created_at, updated_at, account_type)
						VALUES ($1, $2, NOW(), NOW(), 'customer_fbo')
					`, legacyAccountID, namespace)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_sub_account_routes (
							id, namespace, created_at, updated_at,
							routing_key_version, routing_key, account_id, currency
						) VALUES ($1, $2, NOW(), NOW(), 'v1', 'legacy-route', $3, 'USD')
					`, legacyRouteID, namespace, legacyAccountID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (id, namespace, created_at, updated_at)
						VALUES ($1, $2, NOW(), NOW())
					`, legacyGroupID, namespace)
					require.NoError(t, err)
				},
			},
			{
				version:   versionAfter,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// when:
					// - the additive custom-currency migration is applied
					// then:
					// - existing rows remain valid and an old application can keep writing
					var costBasisCurrency sql.NullString
					err := db.QueryRow(`
							SELECT cost_basis_currency
							FROM ledger_sub_account_routes
							WHERE id = $1
						`, legacyRouteID).Scan(&costBasisCurrency)
					require.NoError(t, err)
					require.False(t, costBasisCurrency.Valid)

					_, err = db.Exec(`
						INSERT INTO ledger_sub_account_routes (
							id, namespace, created_at, updated_at,
							routing_key_version, routing_key, account_id, currency
						) VALUES ($1, $2, NOW(), NOW(), 'v1', 'old-writer-route', $3, 'USD')
					`, oldWriterRouteID, namespace, legacyAccountID)
					require.NoError(t, err)

					_, err = db.Exec(`
							INSERT INTO ledger_transaction_groups (id, namespace, created_at, updated_at)
							VALUES ($1, $2, NOW(), NOW())
						`, oldWriterGroupID, namespace)
					require.NoError(t, err)
				},
			},
		},
	}.Test(t)
}
