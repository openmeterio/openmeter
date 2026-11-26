package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	invoiceLineTaxCodeStampPreviousVersion = 20261124154809
	invoiceLineTaxCodeStampVersion         = 20261125224333
)

func TestInvoiceLineTaxCodeStampMigration(t *testing.T) {
	db, migrator := newMigrationTestEnv(t)

	// given: a line with a legacy tax_config and one without, from before the column existed
	require.NoError(t, migrator.Migrate(invoiceLineTaxCodeStampPreviousVersion))

	namespace := "invoice_line_tax_code_stamp"
	_, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	taxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, taxCodeID, "general", "txcd_10000000")

	legacyConfig := `{"behavior": "inclusive", "tax_code_id": "` + taxCodeID + `", "stripe": {"code": "txcd_10000000"}}`
	legacyLineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, legacyLineID, taxCodeID, "inclusive", legacyConfig)
	plainLineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, plainLineID, nil, nil, nil)

	var configBefore string
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT tax_config::text FROM billing_invoice_lines WHERE id = $1`, legacyLineID).Scan(&configBefore))

	// when: migrating up
	require.NoError(t, migrator.Migrate(invoiceLineTaxCodeStampVersion))

	// then: existing rows read as not stamped and the legacy JSON is untouched
	for _, lineID := range []string{legacyLineID, plainLineID} {
		var stamp sql.NullString
		require.NoError(t, db.QueryRowContext(t.Context(),
			`SELECT invoicing_app_tax_code FROM billing_invoice_lines WHERE id = $1`, lineID).Scan(&stamp))
		require.False(t, stamp.Valid)
	}

	var configAfter string
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT tax_config::text FROM billing_invoice_lines WHERE id = $1`, legacyLineID).Scan(&configAfter))
	require.Equal(t, configBefore, configAfter)

	// then: a line with a Tax Code and no tax_config is accepted, as the consistency checks are gone
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, ulid.Make().String(), taxCodeID, "exclusive", nil)

	// when: migrating back down
	require.NoError(t, migrator.Migrate(invoiceLineTaxCodeStampPreviousVersion))

	// then: the column is gone and the rows survive; the old checks are back, but unvalidated so the
	// rows without tax_config do not block the rollback
	var columnCount int
	require.NoError(t, db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_name = 'billing_invoice_lines' AND column_name = 'invoicing_app_tax_code'
	`).Scan(&columnCount))
	require.Zero(t, columnCount)

	var lineCount int
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT count(*) FROM billing_invoice_lines WHERE namespace = $1`, namespace).Scan(&lineCount))
	require.Equal(t, 3, lineCount)
}
