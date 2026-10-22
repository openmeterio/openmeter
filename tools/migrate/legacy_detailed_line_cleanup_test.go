package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const legacyDetailedLineCleanupSeedVersion = 20261021082257

func TestLegacyDetailedLineCleanup(t *testing.T) {
	db, migrator := newMigrationTestEnv(t)
	require.NoError(t, migrator.Migrate(legacyDetailedLineCleanupSeedVersion))

	// Given backfilled invoices, including stale, orphaned and soft-deleted legacy
	// data, and active level-2 details that have been rerated since the backfill.
	namespace := "legacy_detailed_line_cleanup"
	emptyInvoiceID, invoiceID := seedLegacyCleanupInvoiceParents(t, db, namespace)
	parentID, splitID := ulid.Make().String(), ulid.Make().String()
	for _, id := range []string{parentID, splitID} {
		seedGatheringInvoiceLine(t, db, namespace, invoiceID, id, nil, nil, nil)
	}

	feeConfigID, unusedFeeConfigID := ulid.Make().String(), ulid.Make().String()
	usageConfigID, sharedUsageConfigID := ulid.Make().String(), ulid.Make().String()
	_, err := db.ExecContext(t.Context(), `
		UPDATE billing_invoices SET schema_level = CASE WHEN id = $1 THEN 2 ELSE 1 END WHERE id IN ($1, $2)
	`, invoiceID, emptyInvoiceID)
	require.NoError(t, err)
	for _, id := range []string{feeConfigID, unusedFeeConfigID} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_invoice_flat_fee_line_configs (id, namespace, per_unit_amount)
			VALUES ($1, $2, 10)
		`, id, namespace)
		require.NoError(t, err)
	}

	for _, id := range []string{usageConfigID, sharedUsageConfigID} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_invoice_usage_based_line_configs (id, namespace, price_type, price)
			VALUES ($1, $2, 'flat', '{"type":"flat","amount":"10","paymentTerm":"in_advance"}')
		`, id, namespace)
		require.NoError(t, err)
	}

	_, err = db.ExecContext(t.Context(), `
		UPDATE billing_invoice_lines SET type = 'usage_based', usage_based_line_config_id = $2 WHERE id = $1
	`, parentID, sharedUsageConfigID)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET parent_line_id = $1 WHERE id = $2`, parentID, splitID)
	require.NoError(t, err)

	detailedID := seedLegacyCleanupDetailedLine(t, db, namespace, invoiceID)
	orphanID := seedLegacyCleanupDetailedLine(t, db, namespace, invoiceID)
	nestedID := seedLegacyCleanupDetailedLine(t, db, namespace, invoiceID)
	_, err = db.ExecContext(t.Context(), `
		UPDATE billing_invoice_lines
		SET fee_line_config_id = $2, usage_based_line_config_id = $3, parent_line_id = $4,
		    amount = 30, total = 30, metadata = '{"legacy":"original"}', deleted_at = NOW()
		WHERE id = $1
	`, detailedID, feeConfigID, usageConfigID, parentID)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET deleted_at = NOW(), fee_line_config_id = $2 WHERE id = $1`, orphanID, ulid.Make().String())
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET parent_line_id = $1, usage_based_line_config_id = $2 WHERE id = $3`, detailedID, sharedUsageConfigID, nestedID)
	require.NoError(t, err)

	var amountDiscountID string
	for _, lineID := range []string{detailedID, detailedID, parentID} {
		discountID := ulid.Make().String()
		if lineID == detailedID {
			amountDiscountID = discountID
		}

		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_invoice_line_discounts
			    (id, namespace, created_at, updated_at, deleted_at, reason, amount, line_id)
			VALUES ($1, $2, NOW(), NOW(), NOW(), 'maximum_spend', 2, $3)
		`, discountID, namespace+"_legacy", lineID)
		require.NoError(t, err)
	}

	for _, lineID := range []string{detailedID, detailedID, parentID} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_invoice_line_usage_discounts
			    (id, namespace, created_at, updated_at, reason, quantity, line_id)
			VALUES ($1, $2, NOW(), NOW(), 'maximum_spend', 1, $3)
		`, ulid.Make().String(), namespace, lineID)
		require.NoError(t, err)
	}

	seedLegacyCleanupLevel2Detail(t, db, namespace, invoiceID, detailedID, parentID)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_standard_invoice_detailed_line_amount_discounts
		    (id, namespace, created_at, updated_at, reason, amount, line_id)
		VALUES ($1, $2, NOW(), NOW(), 'maximum_spend', 7, $3)
	`, amountDiscountID, namespace, detailedID)
	require.NoError(t, err)

	originalLine := legacyCleanupJSON(t, db, `SELECT to_jsonb(l) FROM billing_invoice_lines l WHERE id = $1`, detailedID)
	originalOrphan := legacyCleanupJSON(t, db, `SELECT to_jsonb(l) FROM billing_invoice_lines l WHERE id = $1`, orphanID)
	originalFeeConfig := legacyCleanupJSON(t, db, `SELECT to_jsonb(c) FROM billing_invoice_flat_fee_line_configs c WHERE id = $1`, feeConfigID)
	originalUsageConfig := legacyCleanupJSON(t, db, `SELECT to_jsonb(c) FROM billing_invoice_usage_based_line_configs c WHERE id = $1`, usageConfigID)
	originalAmounts := legacyCleanupJSON(t, db, `SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM billing_invoice_line_discounts d WHERE line_id = $1`, detailedID)
	originalUsageDiscounts := legacyCleanupJSON(t, db, `SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM billing_invoice_line_usage_discounts d WHERE line_id = $1`, detailedID)
	originalUnusedConfig := legacyCleanupJSON(t, db, `SELECT to_jsonb(c) FROM billing_invoice_flat_fee_line_configs c WHERE id = $1`, unusedFeeConfigID)
	originalStandaloneDiscount := legacyCleanupJSON(t, db, `SELECT to_jsonb(d) FROM billing_invoice_line_discounts d WHERE line_id = $1`, parentID)
	const liveDataQuery = `SELECT jsonb_build_object(
		'lines', (SELECT jsonb_agg(to_jsonb(l) - 'fee_line_config_id' ORDER BY id) FROM billing_invoice_lines l WHERE status <> 'detailed'),
		'details', (SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM billing_standard_invoice_detailed_lines l),
		'amount_discounts', (SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM billing_standard_invoice_detailed_line_amount_discounts d),
		'usage_discounts', (SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM billing_invoice_line_usage_discounts d WHERE line_id = $1),
		'usage_config', (SELECT to_jsonb(c) FROM billing_invoice_usage_based_line_configs c WHERE id = $2),
		'invoices', (SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM billing_invoices i),
		'schema_adapter', (SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM billing_invoice_write_schema_levels s)
	)`
	originalLiveData := legacyCleanupJSON(t, db, liveDataQuery, parentID, sharedUsageConfigID)

	// When the cleanup archives and removes the retired storage.
	require.NoError(t, migrator.Up())

	// Then the full source rows are recoverable and live data is unchanged.
	for field, expected := range map[string]string{
		"line": originalLine, "flat_fee_config": originalFeeConfig,
		"usage_based_config": originalUsageConfig, "amount_discounts": originalAmounts,
		"usage_discounts": originalUsageDiscounts,
	} {
		actual := legacyCleanupJSON(t, db, `
			SELECT row_data -> $2 FROM om_migration_backup_legacy_detailed_lines
			WHERE source_table = 'billing_invoice_lines' AND id = $1
		`, detailedID, field)
		require.JSONEq(t, expected, actual, field)
	}

	require.JSONEq(t, originalOrphan, legacyCleanupJSON(t, db, `SELECT row_data -> 'line' FROM om_migration_backup_legacy_detailed_lines WHERE id = $1`, orphanID))
	require.JSONEq(t, `{"flat_fee_config":null,"usage_based_config":null,"amount_discounts":[],"usage_discounts":[]}`,
		legacyCleanupJSON(t, db, `SELECT row_data - 'line' FROM om_migration_backup_legacy_detailed_lines WHERE id = $1`, orphanID))
	require.JSONEq(t, originalUnusedConfig, legacyCleanupJSON(t, db, `SELECT row_data -> 'flat_fee_config' FROM om_migration_backup_legacy_detailed_lines WHERE id = $1`, unusedFeeConfigID))
	require.JSONEq(t, originalStandaloneDiscount, legacyCleanupJSON(t, db, `SELECT row_data -> 'amount_discount' FROM om_migration_backup_legacy_detailed_lines WHERE source_table = 'billing_invoice_line_discounts'`))
	require.JSONEq(t, originalLiveData, legacyCleanupJSON(t, db, liveDataQuery, parentID, sharedUsageConfigID))

	var archiveCount, detailedCount, obsoleteUsageConfigCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT (SELECT count(*) FROM om_migration_backup_legacy_detailed_lines),
		       (SELECT count(*) FROM billing_invoice_lines WHERE status = 'detailed'),
		       (SELECT count(*) FROM billing_invoice_usage_based_line_configs WHERE id = $1)
	`, usageConfigID).Scan(&archiveCount, &detailedCount, &obsoleteUsageConfigCount)
	require.NoError(t, err)
	require.Equal(t, 5, archiveCount)
	require.Zero(t, detailedCount)
	require.Zero(t, obsoleteUsageConfigCount)
	var retiredStorageRemoved bool
	err = db.QueryRowContext(t.Context(), `SELECT
		to_regclass('billing_invoice_flat_fee_line_configs') IS NULL
		AND to_regclass('billing_invoice_line_discounts') IS NULL
		AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'billing_invoice_lines' AND column_name = 'fee_line_config_id')
		AND to_regprocedure('om_func_migrate_customer_invoices_to_schema_level_2(text)') IS NULL
		AND to_regprocedure('om_func_migrate_customer_invoices_to_schema_level_2_bulk(text[])') IS NULL
	`).Scan(&retiredStorageRemoved)
	require.NoError(t, err)
	require.True(t, retiredStorageRemoved)
}

func TestLegacyDetailedLineCleanupGuards(t *testing.T) {
	const namespace = "legacy_cleanup_guard"

	for _, scenario := range []struct {
		name      string
		setup     func(t *testing.T, db *sql.DB, invoiceID, legacyID string)
		wantError string
	}{
		{
			name: "schema level 1 with lines",
			setup: func(t *testing.T, db *sql.DB, invoiceID, _ string) {
				_, err := db.ExecContext(t.Context(), `UPDATE billing_invoices SET schema_level = 1 WHERE id = $1`, invoiceID)
				require.NoError(t, err)
			},
			wantError: "requires schema level 2",
		},
		{
			name: "live flat fee reference",
			setup: func(t *testing.T, db *sql.DB, _, legacyID string) {
				_, err := db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET status = 'valid', fee_line_config_id = $1 WHERE id = $1`, legacyID)
				require.NoError(t, err)
			},
			wantError: "surviving invoice lines still reference legacy flat-fee",
		},
		{
			name: "live split child",
			setup: func(t *testing.T, db *sql.DB, invoiceID, legacyID string) {
				childID := ulid.Make().String()
				seedGatheringInvoiceLine(t, db, namespace, invoiceID, childID, nil, nil, nil)
				_, err := db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET parent_line_id = $1 WHERE id = $2`, legacyID, childID)
				require.NoError(t, err)
			},
			wantError: "still referenced by billing_invoice_lines.parent_line_id",
		},
		{
			name: "level 2 detail",
			setup: func(t *testing.T, db *sql.DB, invoiceID, legacyID string) {
				seedLegacyCleanupLevel2Detail(t, db, namespace, invoiceID, ulid.Make().String(), legacyID)
			},
			wantError: "still referenced by billing_standard_invoice_detailed_lines.parent_line_id",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, migrator := newMigrationTestEnv(t)
			require.NoError(t, migrator.Migrate(legacyDetailedLineCleanupSeedVersion))

			// Given a legacy line that cannot yet be safely removed.
			_, invoiceID := seedLegacyCleanupInvoiceParents(t, db, namespace)
			legacyID := seedLegacyCleanupDetailedLine(t, db, namespace, invoiceID)
			_, err := db.ExecContext(t.Context(), `UPDATE billing_invoices SET schema_level = 2`)
			require.NoError(t, err)
			scenario.setup(t, db, invoiceID, legacyID)
			const linesQuery = `SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM billing_invoice_lines l`
			originalLines := legacyCleanupJSON(t, db, linesQuery)

			// When cleanup encounters a surviving dependency, then it rolls back.
			err = migrator.Up()
			require.ErrorContains(t, err, scenario.wantError)
			require.JSONEq(t, originalLines, legacyCleanupJSON(t, db, linesQuery))
			var originalSchemaPreserved bool
			err = db.QueryRowContext(t.Context(), `SELECT
				to_regclass('om_migration_backup_legacy_detailed_lines') IS NULL
				AND to_regclass('billing_invoice_flat_fee_line_configs') IS NOT NULL
				AND to_regclass('billing_invoice_line_discounts') IS NOT NULL
			`).Scan(&originalSchemaPreserved)
			require.NoError(t, err)
			require.True(t, originalSchemaPreserved)
		})
	}
}

func seedLegacyCleanupInvoiceParents(t *testing.T, db *sql.DB, namespace string) (string, string) {
	t.Helper()

	gatheringInvoiceID, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	// Billing establishes the customer's lock before creating invoice data.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO billing_customer_locks (id, namespace, customer_id)
		SELECT $1, namespace, customer_id FROM billing_invoices WHERE id = $2
	`, ulid.Make().String(), standardInvoiceID)
	require.NoError(t, err)

	return gatheringInvoiceID, standardInvoiceID
}

func seedLegacyCleanupDetailedLine(t *testing.T, db *sql.DB, namespace, invoiceID string) string {
	t.Helper()

	id := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, invoiceID, id, nil, nil, nil)
	_, err := db.ExecContext(t.Context(), `UPDATE billing_invoice_lines SET status = 'detailed', quantity = 3 WHERE id = $1`, id)
	require.NoError(t, err)

	return id
}

func seedLegacyCleanupLevel2Detail(t *testing.T, db *sql.DB, namespace, invoiceID, id, parentID string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO billing_standard_invoice_detailed_lines (
			id, namespace, created_at, updated_at, name, currency,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total,
			service_period_start, service_period_end, quantity, per_unit_amount, invoice_id, parent_line_id,
			child_unique_reference_id
		) VALUES (
			$1, $2, NOW(), NOW(), 'Rerated detail', 'USD',
			99, 0, 0, 0, 99, 7, 0, 92,
			'2024-01-01', '2024-02-01', 3, 33, $3, $4, $5
		)
	`, id, namespace, invoiceID, parentID, id)
	require.NoError(t, err)
}

func legacyCleanupJSON(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()

	var value string
	err := db.QueryRowContext(t.Context(), query, args...).Scan(&value)
	require.NoError(t, err)

	return value
}
