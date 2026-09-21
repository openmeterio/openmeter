package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	standardInvoiceLineTaxConfigBackfillSeedVersion = 20260921074939
	standardInvoiceLineTaxConfigBackupVersion       = 20260921104045
	standardInvoiceLineTaxConfigBackfillVersion     = 20260921104046
	invoiceLineTaxConsistencyVersion                = 20260921104047
	invoiceLineTaxConsistencyValidateVersion        = 20260921104048
)

func TestStandardInvoiceLineTaxConfigBackfillMigration(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "standard_invoice_line_tax_config_backfill"
	gatheringInvoiceID, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	existingTaxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, existingTaxCodeID, "general", "txcd_10000000")

	embeddedID := ulid.Make().String()
	attachID := ulid.Make().String()
	createID := ulid.Make().String()
	behaviorOnlyID := ulid.Make().String()
	columnsOnlyID := ulid.Make().String()
	noTaxID := ulid.Make().String()
	gatheringControlID := ulid.Make().String()

	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, embeddedID, nil, nil,
		fmt.Sprintf(`{"tax_code_id":%q}`, existingTaxCodeID))
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, attachID, nil, nil,
		`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, createID, nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, behaviorOnlyID, nil, nil,
		`{"behavior":"exclusive"}`)
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, columnsOnlyID, existingTaxCodeID, "inclusive", `null`)
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, noTaxID, nil, nil, nil)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, gatheringControlID, existingTaxCodeID, "exclusive",
		fmt.Sprintf(`{"behavior":"exclusive","tax_code_id":%q}`, existingTaxCodeID))

	_, err := db.ExecContext(t.Context(), `
		UPDATE billing_invoice_lines
		SET deleted_at = NOW()
		WHERE id = $1
	`, columnsOnlyID)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackupVersion))

	var backupCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM om_migration_backup_20260921104045_invoice_line_tax_config
	`).Scan(&backupCount)
	require.NoError(t, err)
	require.Equal(t, 6, backupCount)

	assertStandardInvoiceLineTaxBackup(t, db, embeddedID, []byte(fmt.Sprintf(`{"tax_code_id":%q}`, existingTaxCodeID)), "", "")
	assertStandardInvoiceLineTaxBackup(t, db, attachID, []byte(`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`), "", "")
	assertStandardInvoiceLineTaxBackup(t, db, createID, []byte(`{"stripe":{"code":"txcd_40010001"}}`), "", "")
	assertStandardInvoiceLineTaxBackup(t, db, behaviorOnlyID, []byte(`{"behavior":"exclusive"}`), "", "")
	assertStandardInvoiceLineTaxBackup(t, db, columnsOnlyID, []byte(`null`), existingTaxCodeID, "inclusive")
	assertStandardInvoiceLineTaxBackup(t, db, noTaxID, nil, "", "")

	var gatheringBackupCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM om_migration_backup_20260921104045_invoice_line_tax_config
		WHERE line_id = $1
	`, gatheringControlID).Scan(&gatheringBackupCount)
	require.NoError(t, err)
	require.Zero(t, gatheringBackupCount)

	require.NoError(t, migrator.Migrate(invoiceLineTaxConsistencyValidateVersion))

	assertGatheringInvoiceLineTax(t, db, embeddedID, existingTaxCodeID, "")
	assertGatheringInvoiceLineTax(t, db, attachID, existingTaxCodeID, "inclusive")
	createdTaxCodeID := gatheringInvoiceLineTaxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
	assertGatheringInvoiceLineTax(t, db, createID, createdTaxCodeID, "")
	assertGatheringInvoiceLineTax(t, db, behaviorOnlyID, "", "exclusive")
	assertGatheringInvoiceLineTax(t, db, columnsOnlyID, existingTaxCodeID, "inclusive")
	assertGatheringInvoiceLineTax(t, db, noTaxID, "", "")
	assertGatheringInvoiceLineTax(t, db, gatheringControlID, existingTaxCodeID, "exclusive")

	var deletedAt sql.NullTime
	err = db.QueryRowContext(t.Context(), `
		SELECT deleted_at
		FROM billing_invoice_lines
		WHERE id = $1
	`, columnsOnlyID).Scan(&deletedAt)
	require.NoError(t, err)
	require.True(t, deletedAt.Valid)

	rows, err := db.QueryContext(t.Context(), `
		SELECT conname, convalidated
		FROM pg_constraint
		WHERE conrelid = 'billing_invoice_lines'::regclass
		  AND conname IN (
		    'billing_invoice_line_tax_behavior_consistency',
		    'billing_invoice_line_tax_code_consistency'
		  )
	`)
	require.NoError(t, err)
	defer rows.Close()

	validated := map[string]bool{}
	for rows.Next() {
		var name string
		var isValidated bool
		require.NoError(t, rows.Scan(&name, &isValidated))
		validated[name] = isValidated
	}
	require.NoError(t, rows.Err())
	require.Equal(t, map[string]bool{
		"billing_invoice_line_tax_behavior_consistency": true,
		"billing_invoice_line_tax_code_consistency":     true,
	}, validated)

	_, err = db.ExecContext(t.Context(), `
		UPDATE billing_invoice_lines
		SET tax_behavior = 'inclusive'
		WHERE id = $1
	`, noTaxID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "billing_invoice_line_tax_behavior_consistency")
}

func TestStandardInvoiceLineTaxConfigBackfillMigrationPreservesHistoricalStripeCode(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "standard_invoice_line_tax_config_mismatch"
	_, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	taxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, taxCodeID, "general", "txcd_10000000")
	lineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, lineID, nil, nil,
		fmt.Sprintf(`{"stripe":{"code":"txcd_20060051"},"tax_code_id":%q}`, taxCodeID))

	require.NoError(t, migrator.Migrate(invoiceLineTaxConsistencyValidateVersion))
	assertGatheringInvoiceLineTax(t, db, lineID, taxCodeID, "")

	var stripeCode string
	require.NoError(t, db.QueryRowContext(t.Context(), `
		SELECT tax_config -> 'stripe' ->> 'code'
		FROM billing_invoice_lines
		WHERE id = $1
	`, lineID).Scan(&stripeCode))
	require.Equal(t, "txcd_20060051", stripeCode)
}

func TestStandardInvoiceLineTaxConfigBackupPersistsWhenBackfillFails(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "standard_invoice_line_tax_config_invalid_reference"
	_, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	missingTaxCodeID := ulid.Make().String()
	lineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, lineID, nil, nil,
		fmt.Sprintf(`{"tax_code_id":%q}`, missingTaxCodeID))

	err := migrator.Migrate(standardInvoiceLineTaxConfigBackfillVersion)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or cross-namespace tax code")

	assertStandardInvoiceLineTaxBackup(t, db, lineID, []byte(fmt.Sprintf(`{"tax_code_id":%q}`, missingTaxCodeID)), "", "")
}

func TestStandardInvoiceLineTaxConfigBackfillMigrationAllowsDeletedTaxCodeReference(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "standard_invoice_line_deleted_tax_code"
	_, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	taxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, taxCodeID, "historical", "txcd_10000000")
	lineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, lineID, nil, nil,
		fmt.Sprintf(`{"stripe":{"code":"txcd_10000000"},"tax_code_id":%q}`, taxCodeID))
	stripeOnlyLineID := ulid.Make().String()
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, stripeOnlyLineID, nil, nil,
		`{"stripe":{"code":"txcd_10000000"}}`)

	_, err := db.ExecContext(t.Context(), `
		UPDATE tax_codes
		SET deleted_at = NOW()
		WHERE id = $1
	`, taxCodeID)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(invoiceLineTaxConsistencyValidateVersion))
	assertGatheringInvoiceLineTax(t, db, lineID, taxCodeID, "")
	activeTaxCodeID := gatheringInvoiceLineTaxCodeIDByKey(t, db, namespace, "stripe_txcd_10000000")
	require.NotEqual(t, taxCodeID, activeTaxCodeID)
	assertGatheringInvoiceLineTax(t, db, stripeOnlyLineID, activeTaxCodeID, "")
}

func TestInvoiceLineTaxConsistencyMigrationRevalidatesGatheringRows(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(standardInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "invoice_line_tax_consistency_gathering_recheck"
	gatheringInvoiceID, _ := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, ulid.Make().String(), nil, "inclusive",
		`{"behavior":"exclusive"}`)

	err := migrator.Migrate(invoiceLineTaxConsistencyVersion)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invoice line tax consistency")
	require.Contains(t, err.Error(), "inconsistent tax behavior representations")
}

func assertStandardInvoiceLineTaxBackup(t *testing.T, db *sql.DB, lineID string, wantTaxConfig []byte, wantTaxCodeID, wantTaxBehavior string) {
	t.Helper()

	var taxConfig []byte
	var taxCodeID, taxBehavior sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_config, tax_code_id, tax_behavior
		FROM om_migration_backup_20260921104045_invoice_line_tax_config
		WHERE line_id = $1
		ORDER BY backed_up_at DESC
		LIMIT 1
	`, lineID).Scan(&taxConfig, &taxCodeID, &taxBehavior)
	require.NoError(t, err)

	if wantTaxConfig == nil {
		require.Nil(t, taxConfig)
	} else {
		require.JSONEq(t, string(wantTaxConfig), string(taxConfig))
	}

	if wantTaxCodeID == "" {
		require.False(t, taxCodeID.Valid)
	} else {
		require.Equal(t, wantTaxCodeID, taxCodeID.String)
	}

	if wantTaxBehavior == "" {
		require.False(t, taxBehavior.Valid)
	} else {
		require.Equal(t, wantTaxBehavior, taxBehavior.String)
	}
}
