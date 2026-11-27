package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

const invoiceLineLegacyStripeTaxCodeBackfillVersion = 20261127120359

func TestInvoiceLineLegacyStripeTaxCodeBackfillMigration(t *testing.T) {
	db, migrator := newMigrationTestEnv(t)

	// given: lines with and without a stamp and a legacy stripe code, on a Stripe invoice
	require.NoError(t, migrator.Migrate(invoiceLineTaxCodeStampVersion))

	stripeNamespace := "invoice_line_legacy_stripe_stripe"
	stripeGatheringInvoiceID, stripeInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, stripeNamespace)

	legacyConfig := `{"stripe": {"code": "txcd_10000000"}}`
	blankConfig := `{"stripe": {"code": "  "}}`
	untrimmedConfig := `{"stripe": {"code": " txcd_20000000 "}}`

	unstampedWithCodeID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, legacyConfig, nil)
	unstampedUntrimmedID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, untrimmedConfig, nil)
	unstampedBlankID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, blankConfig, nil)
	unstampedEmptyJSONID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, `{}`, nil)
	unstampedNoConfigID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, nil, nil)
	defaultStampedID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, legacyConfig, lo.ToPtr(""))
	codeStampedID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, legacyConfig, lo.ToPtr("txcd_30000000"))
	deletedUnstampedID := seedLegacyStampedLine(t, db, stripeNamespace, stripeInvoiceID, legacyConfig, nil)

	_, err := db.ExecContext(t.Context(),
		`UPDATE billing_invoice_lines SET deleted_at = NOW() WHERE id = $1`, deletedUnstampedID)
	require.NoError(t, err)

	// given: a line of a gathering invoice that shares the table, which is never stamped
	gatheringID := seedLegacyStampedLine(t, db, stripeNamespace, stripeGatheringInvoiceID, legacyConfig, nil)

	// given: an unstamped line with a legacy stripe code on an invoice of a non-Stripe app
	otherNamespace := "invoice_line_legacy_stripe_other"
	_, otherInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, otherNamespace)

	_, err = db.ExecContext(t.Context(), `UPDATE apps SET type = 'sandbox' WHERE namespace = $1`, otherNamespace)
	require.NoError(t, err)

	nonStripeID := seedLegacyStampedLine(t, db, otherNamespace, otherInvoiceID, legacyConfig, nil)

	lineIDs := []string{
		unstampedWithCodeID, unstampedUntrimmedID, unstampedBlankID, unstampedEmptyJSONID, unstampedNoConfigID,
		defaultStampedID, codeStampedID, deletedUnstampedID, gatheringID, nonStripeID,
	}
	configsBefore := readLineTaxConfigs(t, db, lineIDs)

	// when: migrating up
	require.NoError(t, migrator.Migrate(invoiceLineLegacyStripeTaxCodeBackfillVersion))

	// then: unstamped lines on the Stripe invoice get the trimmed legacy code, deleted lines too
	require.Equal(t, lo.ToPtr("txcd_10000000"), readLineStamp(t, db, unstampedWithCodeID))
	require.Equal(t, lo.ToPtr("txcd_20000000"), readLineStamp(t, db, unstampedUntrimmedID))
	require.Equal(t, lo.ToPtr("txcd_10000000"), readLineStamp(t, db, deletedUnstampedID))

	// then: unstamped lines without a usable legacy code stay NULL
	require.Nil(t, readLineStamp(t, db, unstampedBlankID))
	require.Nil(t, readLineStamp(t, db, unstampedEmptyJSONID))
	require.Nil(t, readLineStamp(t, db, unstampedNoConfigID))

	// then: stamped lines are left alone, "" included
	require.Equal(t, lo.ToPtr(""), readLineStamp(t, db, defaultStampedID))
	require.Equal(t, lo.ToPtr("txcd_30000000"), readLineStamp(t, db, codeStampedID))

	// then: gathering invoice lines and lines of a non-Stripe invoice stay NULL
	require.Nil(t, readLineStamp(t, db, gatheringID))
	require.Nil(t, readLineStamp(t, db, nonStripeID))

	// then: tax_config is untouched everywhere
	require.Equal(t, configsBefore, readLineTaxConfigs(t, db, lineIDs))

	// when: migrating back down
	require.NoError(t, migrator.Migrate(invoiceLineTaxCodeStampVersion))

	// then: the stamps are kept, as the down migration does not repair anything
	require.Equal(t, lo.ToPtr("txcd_10000000"), readLineStamp(t, db, unstampedWithCodeID))
}

// seedLegacyStampedLine inserts a line with the given legacy tax_config JSON (nil for none) and
// invoicing app tax code stamp (nil for NULL), and returns its ID.
func seedLegacyStampedLine(t *testing.T, db *sql.DB, namespace, invoiceID string, taxConfig any, stamp *string) string {
	t.Helper()

	id := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, invoiceID, id, nil, nil, taxConfig)

	if stamp != nil {
		_, err := db.ExecContext(t.Context(),
			`UPDATE billing_invoice_lines SET invoicing_app_tax_code = $1 WHERE id = $2`, *stamp, id)
		require.NoError(t, err)
	}

	return id
}

func readLineStamp(t *testing.T, db *sql.DB, lineID string) *string {
	t.Helper()

	var stamp sql.NullString
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT invoicing_app_tax_code FROM billing_invoice_lines WHERE id = $1`, lineID).Scan(&stamp))

	if !stamp.Valid {
		return nil
	}

	return &stamp.String
}

func readLineTaxConfigs(t *testing.T, db *sql.DB, lineIDs []string) map[string]sql.NullString {
	t.Helper()

	out := make(map[string]sql.NullString, len(lineIDs))

	for _, id := range lineIDs {
		var config sql.NullString
		require.NoError(t, db.QueryRowContext(t.Context(),
			`SELECT tax_config::text FROM billing_invoice_lines WHERE id = $1`, id).Scan(&config))

		out[id] = config
	}

	return out
}
