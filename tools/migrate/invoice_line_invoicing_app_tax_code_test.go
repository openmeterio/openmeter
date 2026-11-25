package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	invoiceLineInvoicingAppTaxCodePreviousVersion = 20261124154809
	invoiceLineInvoicingAppTaxCodeVersion         = 20261125154432
)

func TestInvoiceLineInvoicingAppTaxCodeMigration(t *testing.T) {
	db, migrator := newMigrationTestEnv(t)

	// given: an existing standard line from before the column existed
	require.NoError(t, migrator.Migrate(invoiceLineInvoicingAppTaxCodePreviousVersion))

	namespace := "invoice_line_invoicing_app_tax_code"
	_, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	lineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, lineID, nil, nil, nil)

	// when: migrating up
	require.NoError(t, migrator.Migrate(invoiceLineInvoicingAppTaxCodeVersion))

	// then: the existing row reads as not stamped
	var stamp sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT invoicing_app_tax_code
		FROM billing_invoice_lines
		WHERE id = $1
	`, lineID).Scan(&stamp)
	require.NoError(t, err)
	require.False(t, stamp.Valid)

	// then: the stamp can be set, including the empty string
	_, err = db.ExecContext(t.Context(), `
		UPDATE billing_invoice_lines
		SET invoicing_app_tax_code = ''
		WHERE id = $1
	`, lineID)
	require.NoError(t, err)

	// when: migrating back down
	require.NoError(t, migrator.Migrate(invoiceLineInvoicingAppTaxCodePreviousVersion))

	// then: the column is gone and the row survives
	var columnCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_name = 'billing_invoice_lines' AND column_name = 'invoicing_app_tax_code'
	`).Scan(&columnCount)
	require.NoError(t, err)
	require.Zero(t, columnCount)

	var lineCount int
	err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM billing_invoice_lines WHERE id = $1`, lineID).Scan(&lineCount)
	require.NoError(t, err)
	require.Equal(t, 1, lineCount)
}
