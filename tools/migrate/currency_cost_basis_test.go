package migrate_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestCurrencyCostBasisEffectiveToMigrationPreservesRows(t *testing.T) {
	currencyID := ulid.Make().String()
	costBasisID := ulid.Make().String()
	effectiveFrom := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	up := readMigration(t, "20260701084156_add_currency_cost_basis_effective_to.up.sql")
	require.Contains(t, up, `RENAME COLUMN "custom_currency_id" TO "currency_id"`)
	require.Contains(t, up, `ADD COLUMN "effective_to" timestamptz NULL`)
	require.NotContains(t, up, `DROP COLUMN "custom_currency_id"`)
	require.NotContains(t, up, `ADD COLUMN "currency_id"`)

	down := readMigration(t, "20260701084156_add_currency_cost_basis_effective_to.down.sql")
	require.Contains(t, down, `RENAME COLUMN "currency_id" TO "custom_currency_id"`)
	require.Contains(t, down, `DROP COLUMN "effective_to"`)
	require.NotContains(t, down, `DROP COLUMN "currency_id"`)
	require.NotContains(t, down, `ADD COLUMN "custom_currency_id"`)

	runner{
		stops: stops{
			{
				version:   20260624135146,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`
						INSERT INTO custom_currencies (
							id, namespace, created_at, updated_at, code, name, symbol
						) VALUES (
							$1, 'default', NOW(), NOW(), 'CREDITS', 'Credits', 'CR'
						)
					`, currencyID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO currency_cost_bases (
							id, namespace, created_at, updated_at, fiat_code, rate, effective_from, custom_currency_id
						) VALUES (
							$1, 'default', NOW(), NOW(), 'USD', 0.5, $2, $3
						)
					`, costBasisID, effectiveFrom, currencyID)
					require.NoError(t, err)
				},
			},
			{
				version:   20260701084156,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var gotCurrencyID string
					var effectiveTo sql.NullTime
					err := db.QueryRow(`
						SELECT currency_id, effective_to
						FROM currency_cost_bases
						WHERE id = $1
					`, costBasisID).Scan(&gotCurrencyID, &effectiveTo)
					require.NoError(t, err)
					require.Equal(t, currencyID, gotCurrencyID)
					require.False(t, effectiveTo.Valid)
				},
			},
			{
				version:   20260624135146,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					var gotCurrencyID string
					err := db.QueryRow(`
						SELECT custom_currency_id
						FROM currency_cost_bases
						WHERE id = $1
					`, costBasisID).Scan(&gotCurrencyID)
					require.NoError(t, err)
					require.Equal(t, currencyID, gotCurrencyID)

					var effectiveToExists bool
					err = db.QueryRow(`
						SELECT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'currency_cost_bases'
							  AND column_name = 'effective_to'
						)
					`).Scan(&effectiveToExists)
					require.NoError(t, err)
					require.False(t, effectiveToExists)
				},
			},
		},
	}.Test(t)
}

func readMigration(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("migrations", name))
	require.NoError(t, err)

	return string(data)
}
