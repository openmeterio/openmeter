package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

const chargeDiscountCorrelationIDMigration = "20260809152600_backfill_charge_discount_correlation_ids.up.sql"

func TestBackfillChargeDiscountCorrelationIDs(t *testing.T) {
	up := readMigration(t, chargeDiscountCorrelationIDMigration)

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()

	conn, err := testDB.PGDriver.DB().Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	createChargeDiscountCorrelationIDMigrationFixtures(t, conn)

	untouchedBefore := readUntouchedDiscounts(t, conn)

	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)

	percentageID := assertGeneratedGatheringLineDiscountCorrelationID(t, conn, "charge-both", "percentage")
	usageID := assertGeneratedGatheringLineDiscountCorrelationID(t, conn, "charge-both", "usage")
	require.NotEqual(t, percentageID, usageID)

	assertGatheringLineDiscountCorrelationID(t, conn, "charge-existing", "percentage", "existing-percentage")
	assertGeneratedGatheringLineDiscountCorrelationID(t, conn, "charge-existing", "usage")
	require.Equal(t, untouchedBefore, readUntouchedDiscounts(t, conn))

	afterFirstRun := readGatheringLineDiscounts(t, conn)
	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)
	require.Equal(t, afterFirstRun, readGatheringLineDiscounts(t, conn))
}

func createChargeDiscountCorrelationIDMigrationFixtures(t *testing.T, conn *sql.Conn) {
	t.Helper()

	_, err := conn.ExecContext(t.Context(), `
		CREATE SEQUENCE migration_test_ulid_sequence;
		CREATE FUNCTION om_func_generate_ulid()
		RETURNS text
		AS $$
			SELECT 'generated-' || nextval('migration_test_ulid_sequence')::text
		$$
		LANGUAGE sql
		VOLATILE;

		CREATE TEMP TABLE billing_gathering_invoice_lines (
			id text PRIMARY KEY,
			charge_id text NULL,
			ratecard_discounts jsonb NULL,
			updated_at timestamptz NOT NULL DEFAULT '2026-08-01T00:00:00Z',
			deleted_at timestamptz NULL
		);
		CREATE TEMP TABLE billing_invoice_lines (
			id text PRIMARY KEY,
			charge_id text NULL,
			ratecard_discounts jsonb NULL,
			deleted_at timestamptz NULL
		);
		CREATE TEMP TABLE charge_usage_based (
			id text PRIMARY KEY,
			discounts jsonb NULL
		);

		INSERT INTO billing_gathering_invoice_lines (id, charge_id, ratecard_discounts, deleted_at) VALUES
			('charge-both', 'charge-1', '{
				"percentage":{"percentage":10},
				"usage":{"quantity":"5","correlationID":""}
			}', NULL),
			('charge-existing', 'charge-2', '{
				"percentage":{"percentage":15,"correlationID":"existing-percentage"},
				"usage":{"quantity":"3"}
			}', NULL),
			('not-charge-backed', NULL, '{
				"percentage":{"percentage":20},
				"usage":{"quantity":"7"}
			}', NULL),
			('deleted-charge-backed', 'charge-3', '{
				"percentage":{"percentage":25}
			}', '2026-08-03T01:00:00Z'),
			('without-discounts', 'charge-4', NULL, NULL);

		INSERT INTO billing_invoice_lines (id, charge_id, ratecard_discounts, deleted_at) VALUES
			('standard-charge-backed', 'charge-1', '{
				"percentage":{"percentage":10},
				"usage":{"quantity":"5"}
			}', NULL);

		INSERT INTO charge_usage_based (id, discounts) VALUES
			('charge-1', '{
				"percentage":{"percentage":10},
				"usage":{"quantity":"5"}
			}');
	`)
	require.NoError(t, err)
}

func assertGatheringLineDiscountCorrelationID(t *testing.T, conn *sql.Conn, id, discountType, expected string) {
	t.Helper()

	var actual string
	err := conn.QueryRowContext(t.Context(),
		`SELECT ratecard_discounts #>> ARRAY[$1, 'correlationID'] FROM billing_gathering_invoice_lines WHERE id = $2`,
		discountType,
		id,
	).Scan(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func assertGeneratedGatheringLineDiscountCorrelationID(t *testing.T, conn *sql.Conn, id, discountType string) string {
	t.Helper()

	var actual string
	err := conn.QueryRowContext(t.Context(),
		`SELECT ratecard_discounts #>> ARRAY[$1, 'correlationID'] FROM billing_gathering_invoice_lines WHERE id = $2`,
		discountType,
		id,
	).Scan(&actual)
	require.NoError(t, err)
	require.Contains(t, actual, "generated-")

	return actual
}

func readUntouchedDiscounts(t *testing.T, conn *sql.Conn) map[string]string {
	t.Helper()

	rows, err := conn.QueryContext(t.Context(), `
		SELECT 'gathering:' || id, COALESCE(ratecard_discounts::text, 'null')
		FROM billing_gathering_invoice_lines
		WHERE charge_id IS NULL OR deleted_at IS NOT NULL OR ratecard_discounts IS NULL
		UNION ALL
		SELECT 'standard:' || id, COALESCE(ratecard_discounts::text, 'null')
		FROM billing_invoice_lines
		UNION ALL
		SELECT 'charge:' || id, COALESCE(discounts::text, 'null')
		FROM charge_usage_based
	`)
	require.NoError(t, err)
	defer rows.Close()

	return scanDiscounts(t, rows)
}

func readGatheringLineDiscounts(t *testing.T, conn *sql.Conn) map[string]string {
	t.Helper()

	rows, err := conn.QueryContext(t.Context(), `
		SELECT id, COALESCE(ratecard_discounts::text, 'null')
		FROM billing_gathering_invoice_lines
	`)
	require.NoError(t, err)
	defer rows.Close()

	return scanDiscounts(t, rows)
}

func scanDiscounts(t *testing.T, rows *sql.Rows) map[string]string {
	t.Helper()

	result := map[string]string{}
	for rows.Next() {
		var key, discounts string
		require.NoError(t, rows.Scan(&key, &discounts))
		result[key] = discounts
	}
	require.NoError(t, rows.Err())

	return result
}
