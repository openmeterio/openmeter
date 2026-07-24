package migrate_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLedgerCustomCurrencyMigrationBackwardCompatibility(t *testing.T) {
	const (
		versionBefore = 20260724082957
		versionAfter  = 20260724112327
		namespace     = "tenant-a"
		sharedKey     = "shared-key"
	)

	legacyAccountID := ulid.Make().String()
	legacyRouteID := ulid.Make().String()
	oldWriterRouteID := ulid.Make().String()
	legacyGroupID := ulid.Make().String()
	oldWriterGroupID := ulid.Make().String()
	firstKeyedGroupID := ulid.Make().String()
	secondKeyedGroupID := ulid.Make().String()
	fingerprint := "v1:" + strings.Repeat("a", 64)

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
					var exchangeSource sql.NullString
					err := db.QueryRow(`
						SELECT exchange_source_currency
						FROM ledger_sub_account_routes
						WHERE id = $1
					`, legacyRouteID).Scan(&exchangeSource)
					require.NoError(t, err)
					require.False(t, exchangeSource.Valid)

					var (
						idempotencyScope sql.NullString
						idempotencyKey   sql.NullString
						inputFingerprint sql.NullString
					)
					err = db.QueryRow(`
						SELECT idempotency_scope, idempotency_key, input_fingerprint
						FROM ledger_transaction_groups
						WHERE id = $1
					`, legacyGroupID).Scan(&idempotencyScope, &idempotencyKey, &inputFingerprint)
					require.NoError(t, err)
					require.False(t, idempotencyScope.Valid)
					require.False(t, idempotencyKey.Valid)
					require.False(t, inputFingerprint.Valid)

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

					// Namespace-scoped keyed writes remain distinct while exact replays are fenced.
					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (
							id, namespace, created_at, updated_at,
							idempotency_scope, idempotency_key, input_fingerprint
						) VALUES ($1, $2, NOW(), NOW(), $3, $4, $5)
					`, firstKeyedGroupID, namespace, "8:tenant-a"+sharedKey, sharedKey, fingerprint)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (
							id, namespace, created_at, updated_at,
							idempotency_scope, idempotency_key, input_fingerprint
						) VALUES ($1, 'tenant-b', NOW(), NOW(), $2, $3, $4)
					`, secondKeyedGroupID, "8:tenant-b"+sharedKey, sharedKey, fingerprint)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (
							id, namespace, created_at, updated_at,
							idempotency_scope, idempotency_key, input_fingerprint
						) VALUES ($1, $2, NOW(), NOW(), $3, $4, $5)
					`, ulid.Make().String(), namespace, "8:tenant-a"+sharedKey, sharedKey, fingerprint)
					require.Error(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (
							id, namespace, created_at, updated_at,
							idempotency_scope, idempotency_key, input_fingerprint
						) VALUES ($1, $2, NOW(), NOW(), $3, $4, $5)
					`, ulid.Make().String(), namespace, "incorrect-scope", "different-key", fingerprint)
					require.Error(t, err)

					_, err = db.Exec(`
						INSERT INTO ledger_transaction_groups (
							id, namespace, created_at, updated_at,
							idempotency_key, input_fingerprint
						) VALUES ($1, $2, NOW(), NOW(), $3, $4)
					`, ulid.Make().String(), namespace, "missing-scope", fingerprint)
					require.Error(t, err)
				},
			},
		},
	}.Test(t)
}
