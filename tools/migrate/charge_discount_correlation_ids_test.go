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

	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)

	assertDiscountCorrelationID(t, conn, "charge_usage_based", "usage-line", "percentage", "gathering-percentage")
	assertDiscountCorrelationID(t, conn, "charge_usage_based", "usage-line", "usage", "gathering-usage")
	assertDiscountCorrelationID(t, conn, "charge_usage_based", "usage-existing", "percentage", "existing-percentage")
	assertDiscountCorrelationID(t, conn, "charge_usage_based", "usage-existing", "usage", "existing-usage")
	assertGeneratedDiscountCorrelationID(t, conn, "charge_usage_based", "usage-generated", "usage")

	assertDiscountCorrelationID(t, conn, "charge_usage_based_overrides", "usage-override", "percentage", "gathering-percentage")
	assertDiscountCorrelationID(t, conn, "charge_usage_based_overrides", "usage-override", "usage", "override-existing-usage")
	assertGeneratedDiscountCorrelationID(t, conn, "charge_usage_based_overrides", "usage-generated-override", "usage")

	assertDiscountCorrelationID(t, conn, "charge_flat_fees", "flat-line", "percentage", "standard-percentage")
	assertDiscountCorrelationID(t, conn, "charge_flat_fees", "flat-existing", "percentage", "existing-flat-percentage")
	assertGeneratedDiscountCorrelationID(t, conn, "charge_flat_fee_overrides", "flat-generated-override", "percentage")

	var nullDiscounts bool
	err = conn.QueryRowContext(t.Context(), `SELECT discounts IS NULL FROM charge_flat_fees WHERE id = 'flat-without-discount'`).Scan(&nullDiscounts)
	require.NoError(t, err)
	require.True(t, nullDiscounts)

	beforeRetry := readChargeDiscounts(t, conn)
	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)
	require.Equal(t, beforeRetry, readChargeDiscounts(t, conn))
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
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL
		);
		CREATE TEMP TABLE billing_invoice_lines (
			id text PRIMARY KEY,
			charge_id text NULL,
			ratecard_discounts jsonb NULL,
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL
		);
		CREATE TEMP TABLE charge_usage_based (
			id text PRIMARY KEY,
			discounts jsonb NULL,
			updated_at timestamptz NOT NULL
		);
		CREATE TEMP TABLE charge_usage_based_overrides (
			id text PRIMARY KEY,
			charge_id text NOT NULL,
			discounts jsonb NULL
		);
		CREATE TEMP TABLE charge_flat_fees (
			id text PRIMARY KEY,
			discounts jsonb NULL,
			updated_at timestamptz NOT NULL
		);
		CREATE TEMP TABLE charge_flat_fee_overrides (
			id text PRIMARY KEY,
			charge_id text NOT NULL,
			discounts jsonb NULL
		);

		INSERT INTO billing_gathering_invoice_lines (id, charge_id, ratecard_discounts, updated_at, deleted_at) VALUES
			('usage-gathering', 'usage-line', '{
				"percentage":{"percentage":10,"correlationID":"gathering-percentage"},
				"usage":{"quantity":"5","correlationID":"gathering-usage"}
			}', '2026-08-01T00:00:00Z', NULL),
			('flat-deleted-gathering', 'flat-line', '{
				"percentage":{"percentage":20,"correlationID":"deleted-gathering-percentage"}
			}', '2026-08-03T00:00:00Z', '2026-08-03T01:00:00Z');

		INSERT INTO billing_invoice_lines (id, charge_id, ratecard_discounts, updated_at, deleted_at) VALUES
			('usage-standard', 'usage-line', '{
				"percentage":{"percentage":10,"correlationID":"standard-percentage"},
				"usage":{"quantity":"5","correlationID":"standard-usage"}
			}', '2026-08-02T00:00:00Z', NULL),
			('flat-standard', 'flat-line', '{
				"percentage":{"percentage":20,"correlationID":"standard-percentage"}
			}', '2026-08-02T00:00:00Z', NULL);

		INSERT INTO charge_usage_based (id, discounts, updated_at) VALUES
			('usage-line', '{
				"percentage":{"percentage":10},
				"usage":{"quantity":"5","correlationID":""}
			}', '2026-08-01T00:00:00Z'),
			('usage-existing', '{
				"percentage":{"percentage":15,"correlationID":"existing-percentage"},
				"usage":{"quantity":"3","correlationID":"existing-usage"}
			}', '2026-08-01T00:00:00Z'),
			('usage-generated', '{
				"usage":{"quantity":"8"}
			}', '2026-08-01T00:00:00Z');

		INSERT INTO charge_usage_based_overrides (id, charge_id, discounts) VALUES
			('usage-override', 'usage-line', '{
				"percentage":{"percentage":25},
				"usage":{"quantity":"7","correlationID":"override-existing-usage"}
			}'),
			('usage-generated-override', 'usage-generated', '{
				"usage":{"quantity":"9","correlationID":""}
			}');

		INSERT INTO charge_flat_fees (id, discounts, updated_at) VALUES
			('flat-line', '{"percentage":{"percentage":20}}', '2026-08-01T00:00:00Z'),
			('flat-existing', '{"percentage":{"percentage":30,"correlationID":"existing-flat-percentage"}}', '2026-08-01T00:00:00Z'),
			('flat-without-discount', NULL, '2026-08-01T00:00:00Z');

		INSERT INTO charge_flat_fee_overrides (id, charge_id, discounts) VALUES
			('flat-generated-override', 'flat-without-discount', '{"percentage":{"percentage":40,"correlationID":""}}');
	`)
	require.NoError(t, err)
}

func assertDiscountCorrelationID(t *testing.T, conn *sql.Conn, table, id, discountType, expected string) {
	t.Helper()

	var actual string
	err := conn.QueryRowContext(t.Context(),
		`SELECT discounts #>> ARRAY[$1, 'correlationID'] FROM `+table+` WHERE id = $2`,
		discountType,
		id,
	).Scan(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func assertGeneratedDiscountCorrelationID(t *testing.T, conn *sql.Conn, table, id, discountType string) {
	t.Helper()

	var actual string
	err := conn.QueryRowContext(t.Context(),
		`SELECT discounts #>> ARRAY[$1, 'correlationID'] FROM `+table+` WHERE id = $2`,
		discountType,
		id,
	).Scan(&actual)
	require.NoError(t, err)
	require.NotEmpty(t, actual)
	require.Contains(t, actual, "generated-")
}

func readChargeDiscounts(t *testing.T, conn *sql.Conn) map[string]string {
	t.Helper()

	rows, err := conn.QueryContext(t.Context(), `
		SELECT 'usage:' || id, COALESCE(discounts::text, 'null') FROM charge_usage_based
		UNION ALL
		SELECT 'usage_override:' || id, COALESCE(discounts::text, 'null') FROM charge_usage_based_overrides
		UNION ALL
		SELECT 'flat:' || id, COALESCE(discounts::text, 'null') FROM charge_flat_fees
		UNION ALL
		SELECT 'flat_override:' || id, COALESCE(discounts::text, 'null') FROM charge_flat_fee_overrides
	`)
	require.NoError(t, err)
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var key, discounts string
		require.NoError(t, rows.Scan(&key, &discounts))
		result[key] = discounts
	}
	require.NoError(t, rows.Err())

	return result
}
