package migrate_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

const creditPurchaseCostBasisSchemaLevel2Migration = "20260807061955_migrate_credit_purchase_cost_basis_schema_level_2.up.sql"

func TestMigrateCreditPurchaseCostBasisSchemaLevel2(t *testing.T) {
	withCreditPurchaseCostBasisBackfillTables(t, func(db *sql.DB) {
		createdAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

		_, err := db.ExecContext(t.Context(), `
			INSERT INTO charge_credit_purchases (
				id, namespace, created_at, currency, custom_currency_id, settlement, schema_level,
				fiat_cost_basis, settlement_type, initial_payment_settlement_status
			) VALUES
				('promotional', 'default', $1, 'USD', NULL, '{"type":"promotional"}', 1, NULL, NULL, NULL),
				('invoice-fiat', 'default', $1, 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"0.5"}', 1, NULL, NULL, NULL),
				('external-custom', 'default', $1, NULL, 'custom-1', '{"type":"external","currency":"EUR","costBasis":"1.25","status":"authorized"}', 1, NULL, NULL, NULL),
				('native-level-2', 'default', $1, 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"2"}', 2, 2, 'invoice', NULL)
		`, createdAt)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), readMigration(t, creditPurchaseCostBasisSchemaLevel2Migration))
		require.NoError(t, err)

		var schemaLevel int
		var settlementType string
		var fiatCostBasis, costBasisID, initialStatus sql.NullString

		err = db.QueryRowContext(t.Context(), `
			SELECT schema_level, settlement_type, fiat_cost_basis::text, cost_basis_id, initial_payment_settlement_status
			FROM charge_credit_purchases
			WHERE id = 'promotional'
		`).Scan(&schemaLevel, &settlementType, &fiatCostBasis, &costBasisID, &initialStatus)
		require.NoError(t, err)
		require.Equal(t, 2, schemaLevel)
		require.Equal(t, "promotional", settlementType)
		require.False(t, fiatCostBasis.Valid)
		require.False(t, costBasisID.Valid)
		require.False(t, initialStatus.Valid)

		err = db.QueryRowContext(t.Context(), `
			SELECT schema_level, settlement_type, fiat_cost_basis::text, cost_basis_id, initial_payment_settlement_status
			FROM charge_credit_purchases
			WHERE id = 'invoice-fiat'
		`).Scan(&schemaLevel, &settlementType, &fiatCostBasis, &costBasisID, &initialStatus)
		require.NoError(t, err)
		require.Equal(t, 2, schemaLevel)
		require.Equal(t, "invoice", settlementType)
		require.Equal(t, "0.5", fiatCostBasis.String)
		require.False(t, costBasisID.Valid)
		require.False(t, initialStatus.Valid)

		err = db.QueryRowContext(t.Context(), `
			SELECT schema_level, settlement_type, fiat_cost_basis::text, cost_basis_id, initial_payment_settlement_status
			FROM charge_credit_purchases
			WHERE id = 'external-custom'
		`).Scan(&schemaLevel, &settlementType, &fiatCostBasis, &costBasisID, &initialStatus)
		require.NoError(t, err)
		require.Equal(t, 2, schemaLevel)
		require.Equal(t, "external", settlementType)
		require.False(t, fiatCostBasis.Valid)
		require.True(t, costBasisID.Valid)
		require.Equal(t, "authorized", initialStatus.String)

		var mode, fiatCurrency, manualRate, resolvedCostBasis, namespace, currencyID string
		var resolvedAt time.Time
		var currencyCostBasisID, resolvedCostBasisID sql.NullString
		err = db.QueryRowContext(t.Context(), `
			SELECT
				mode, fiat_currency, manual_rate::text, resolved_cost_basis::text, resolved_at,
				namespace, currency_id, currency_cost_basis_id, resolved_cost_basis_id
			FROM charge_credit_purchase_cost_bases
			WHERE id = $1
		`, costBasisID.String).Scan(
			&mode,
			&fiatCurrency,
			&manualRate,
			&resolvedCostBasis,
			&resolvedAt,
			&namespace,
			&currencyID,
			&currencyCostBasisID,
			&resolvedCostBasisID,
		)
		require.NoError(t, err)
		require.Equal(t, "manual", mode)
		require.Equal(t, "EUR", fiatCurrency)
		require.Equal(t, "1.25", manualRate)
		require.Equal(t, "1.25", resolvedCostBasis)
		require.Equal(t, createdAt, resolvedAt.UTC())
		require.Equal(t, "default", namespace)
		require.Equal(t, "custom-1", currencyID)
		require.False(t, currencyCostBasisID.Valid)
		require.False(t, resolvedCostBasisID.Valid)

		var nativeFiatCostBasis string
		err = db.QueryRowContext(t.Context(), `
			SELECT schema_level, fiat_cost_basis::text
			FROM charge_credit_purchases
			WHERE id = 'native-level-2'
		`).Scan(&schemaLevel, &nativeFiatCostBasis)
		require.NoError(t, err)
		require.Equal(t, 2, schemaLevel)
		require.Equal(t, "2", nativeFiatCostBasis)

		var legacyRows int
		err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM charge_credit_purchases WHERE schema_level = 1`).Scan(&legacyRows)
		require.NoError(t, err)
		require.Zero(t, legacyRows)
	})
}

func TestMigrateCreditPurchaseCostBasisSchemaLevel2WaitsForLockedRows(t *testing.T) {
	withCreditPurchaseCostBasisBackfillTables(t, func(db *sql.DB) {
		_, err := db.ExecContext(t.Context(), `
			INSERT INTO charge_credit_purchases (
				id, namespace, created_at, currency, settlement, schema_level
			) VALUES (
				'locked-row', 'default', NOW(), 'USD',
				'{"type":"invoice","currency":"USD","costBasis":"1"}', 1
			)
		`)
		require.NoError(t, err)

		lockTx, err := db.BeginTx(t.Context(), nil)
		require.NoError(t, err)
		defer func() {
			_ = lockTx.Rollback()
		}()

		_, err = lockTx.ExecContext(t.Context(), `
			SELECT id
			FROM charge_credit_purchases
			WHERE id = 'locked-row'
			FOR UPDATE
		`)
		require.NoError(t, err)

		migrationConn, err := db.Conn(t.Context())
		require.NoError(t, err)
		defer migrationConn.Close()

		migrationContext, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		_, err = migrationConn.ExecContext(migrationContext, readMigration(t, creditPurchaseCostBasisSchemaLevel2Migration))
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded)

		require.NoError(t, lockTx.Rollback())

		var schemaLevel int
		err = db.QueryRowContext(t.Context(), `SELECT schema_level FROM charge_credit_purchases WHERE id = 'locked-row'`).Scan(&schemaLevel)
		require.NoError(t, err)
		require.Equal(t, 1, schemaLevel)
	})
}

func TestMigrateCreditPurchaseCostBasisSchemaLevel2RejectsInvalidRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		insert  string
		wantErr string
	}{
		{
			name:    "unsupported settlement type",
			insert:  `('invalid-type', 'default', NOW(), 'USD', NULL, '{"type":"unknown"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "unsupported settlement types",
		},
		{
			name:    "dedicated state on legacy row",
			insert:  `('dedicated-state', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"1"}', 1, 1, NULL, NULL, NULL)`,
			wantErr: "dedicated schema-level state",
		},
		{
			name:    "missing cost basis",
			insert:  `('missing-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "zero cost basis",
			insert:  `('zero-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"0"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "negative cost basis",
			insert:  `('negative-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"-1"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "NaN cost basis",
			insert:  `('nan-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"NaN"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "infinite cost basis",
			insert:  `('infinite-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"Infinity"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "malformed cost basis",
			insert:  `('malformed-rate', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"USD","costBasis":"not-a-number"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid cost basis",
		},
		{
			name:    "mismatched fiat settlement currency",
			insert:  `('currency-mismatch', 'default', NOW(), 'USD', NULL, '{"type":"invoice","currency":"EUR","costBasis":"1"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid settlement currency",
		},
		{
			name:    "missing custom currency",
			insert:  `('missing-custom-currency', 'default', NOW(), NULL, NULL, '{"type":"invoice","currency":"EUR","costBasis":"1"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid settlement currency",
		},
		{
			name:    "missing external status",
			insert:  `('missing-status', 'default', NOW(), 'USD', NULL, '{"type":"external","currency":"USD","costBasis":"1"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid settlement compatibility fields",
		},
		{
			name:    "promotional payment fields",
			insert:  `('promotional-fields', 'default', NOW(), 'USD', NULL, '{"type":"promotional","currency":"USD"}', 1, NULL, NULL, NULL, NULL)`,
			wantErr: "invalid settlement compatibility fields",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withCreditPurchaseCostBasisBackfillTables(t, func(db *sql.DB) {
				_, err := db.ExecContext(t.Context(), `
					INSERT INTO charge_credit_purchases (
						id, namespace, created_at, currency, custom_currency_id, settlement, schema_level,
						fiat_cost_basis, cost_basis_id, settlement_type, initial_payment_settlement_status
					) VALUES `+tc.insert)
				require.NoError(t, err)

				_, err = db.ExecContext(t.Context(), readMigration(t, creditPurchaseCostBasisSchemaLevel2Migration))
				require.ErrorContains(t, err, tc.wantErr)
			})
		})
	}
}

func withCreditPurchaseCostBasisBackfillTables(t *testing.T, action func(db *sql.DB)) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()

	db := testDB.PGDriver.DB()
	_, err := db.ExecContext(t.Context(), `
		CREATE SEQUENCE om_test_ulid_sequence;

		CREATE FUNCTION om_func_generate_ulid()
		RETURNS text
		LANGUAGE sql
		VOLATILE
		AS $$
			SELECT lpad(nextval('om_test_ulid_sequence')::text, 26, '0')
		$$;

		CREATE TABLE charge_credit_purchases (
			id text PRIMARY KEY,
			namespace text NOT NULL,
			created_at timestamptz NOT NULL,
			currency varchar(3) NULL,
			custom_currency_id text NULL,
			settlement jsonb NOT NULL,
			schema_level smallint NOT NULL,
			fiat_cost_basis numeric NULL,
			settlement_type text NULL,
			initial_payment_settlement_status text NULL,
			cost_basis_id text NULL
		);

		CREATE TABLE charge_credit_purchase_cost_bases (
			id text PRIMARY KEY,
			mode text NOT NULL,
			fiat_currency varchar(3) NOT NULL,
			manual_rate numeric NULL,
			resolved_cost_basis numeric NULL,
			resolved_at timestamptz NULL,
			namespace text NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL,
			currency_cost_basis_id text NULL,
			resolved_cost_basis_id text NULL,
			currency_id text NOT NULL
		);
	`)
	require.NoError(t, err)

	action(db)
}
