package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	gatheringInvoiceLineTaxConfigBackfillSeedVersion = 20260917150829
	gatheringInvoiceLineTaxConfigBackfillVersion     = 20260921074939
)

func TestGatheringInvoiceLineTaxConfigBackfillMigration(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(gatheringInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "gathering_invoice_line_tax_config_backfill"
	gatheringInvoiceID, standardInvoiceID := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	existingTaxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, existingTaxCodeID, "general", "txcd_10000000")

	attachID := ulid.Make().String()
	createID := ulid.Make().String()
	behaviorOnlyID := ulid.Make().String()
	columnsOnlyID := ulid.Make().String()
	noTaxID := ulid.Make().String()
	standardID := ulid.Make().String()

	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, attachID, nil, nil,
		`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, createID, nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, behaviorOnlyID, nil, nil,
		`{"behavior":"exclusive"}`)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, columnsOnlyID, existingTaxCodeID, "inclusive", `null`)
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, noTaxID, nil, nil, nil)
	seedGatheringInvoiceLine(t, db, namespace, standardInvoiceID, standardID, nil, nil,
		`{"stripe":{"code":"txcd_20060051"}}`)

	require.NoError(t, migrator.Migrate(gatheringInvoiceLineTaxConfigBackfillVersion))

	assertGatheringInvoiceLineTax(t, db, attachID, existingTaxCodeID, "inclusive")
	createdTaxCodeID := gatheringInvoiceLineTaxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
	assertGatheringInvoiceLineTax(t, db, createID, createdTaxCodeID, "")
	assertGatheringInvoiceLineTax(t, db, behaviorOnlyID, "", "exclusive")
	assertGatheringInvoiceLineTax(t, db, columnsOnlyID, existingTaxCodeID, "inclusive")
	assertGatheringInvoiceLineTax(t, db, noTaxID, "", "")

	var standardTaxCodeID, embeddedStandardTaxCodeID sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_code_id, tax_config ->> 'tax_code_id'
		FROM billing_invoice_lines
		WHERE id = $1
	`, standardID).Scan(&standardTaxCodeID, &embeddedStandardTaxCodeID)
	require.NoError(t, err)
	require.False(t, standardTaxCodeID.Valid)
	require.False(t, embeddedStandardTaxCodeID.Valid)

	var standardOnlyTaxCodeCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM tax_codes
		WHERE namespace = $1
		  AND key = 'stripe_txcd_20060051'
		  AND deleted_at IS NULL
	`, namespace).Scan(&standardOnlyTaxCodeCount)
	require.NoError(t, err)
	require.Zero(t, standardOnlyTaxCodeCount)
}

func TestGatheringInvoiceLineTaxConfigBackfillMigrationFailsOnMismatchedTaxIdentity(t *testing.T) {
	db, migrator := newGatheringInvoiceLineTaxConfigBackfillTestEnv(t)
	require.NoError(t, migrator.Migrate(gatheringInvoiceLineTaxConfigBackfillSeedVersion))

	namespace := "gathering_invoice_line_tax_config_mismatch"
	gatheringInvoiceID, _ := seedGatheringInvoiceLineTaxParents(t, db, namespace)
	taxCodeID := ulid.Make().String()
	seedGatheringInvoiceLineTaxCode(t, db, namespace, taxCodeID, "general", "txcd_10000000")
	seedGatheringInvoiceLine(t, db, namespace, gatheringInvoiceID, ulid.Make().String(), nil, nil,
		fmt.Sprintf(`{"stripe":{"code":"txcd_20060051"},"tax_code_id":%q}`, taxCodeID))

	err := migrator.Migrate(gatheringInvoiceLineTaxConfigBackfillVersion)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match the referenced live tax code Stripe app mapping")
}

func newGatheringInvoiceLineTaxConfigBackfillTestEnv(t *testing.T) (*sql.DB, *migrate.Migrate) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() {
		_ = testDB.PGDriver.Close()
	})

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = migrator.Close()
	})

	return testDB.PGDriver.DB(), migrator
}

func seedGatheringInvoiceLineTaxParents(t *testing.T, db *sql.DB, namespace string) (string, string) {
	t.Helper()

	customerID := ulid.Make().String()
	taxAppID := ulid.Make().String()
	invoicingAppID := ulid.Make().String()
	paymentAppID := ulid.Make().String()
	profileWorkflowID := ulid.Make().String()
	invoiceWorkflowID := ulid.Make().String()
	profileID := ulid.Make().String()
	gatheringInvoiceID := ulid.Make().String()
	standardInvoiceID := ulid.Make().String()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO customers (
			id, namespace, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, NOW(), NOW(), 'tax-migration-customer', 'Tax migration customer', 'USD'
		)
	`, customerID, namespace)
	require.NoError(t, err)

	for _, app := range []struct {
		id   string
		name string
	}{
		{id: taxAppID, name: "Tax app"},
		{id: invoicingAppID, name: "Invoicing app"},
		{id: paymentAppID, name: "Payment app"},
	} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO apps (id, namespace, created_at, updated_at, name, type, status)
			VALUES ($1, $2, NOW(), NOW(), $3, 'stripe', 'active')
		`, app.id, namespace, app.name)
		require.NoError(t, err)
	}

	for _, workflowID := range []string{profileWorkflowID, invoiceWorkflowID} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_workflow_configs (
				id, namespace, created_at, updated_at,
				collection_alignment, line_collection_period,
				invoice_auto_advance, invoice_draft_period,
				invoice_due_after, invoice_collection_method,
				invoice_progressive_billing
			) VALUES (
				$1, $2, NOW(), NOW(),
				'subscription', 'P1M', true, 'P1D',
				'P30D', 'charge_automatically', false
			)
		`, workflowID, namespace)
		require.NoError(t, err)
	}

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_profiles (
			id, namespace, created_at, updated_at,
			name, supplier_name, "default",
			tax_app_id, invoicing_app_id, payment_app_id, workflow_config_id
		) VALUES (
			$1, $2, NOW(), NOW(),
			'Tax migration profile', 'Tax migration supplier', true,
			$3, $4, $5, $6
		)
	`, profileID, namespace, taxAppID, invoicingAppID, paymentAppID, profileWorkflowID)
	require.NoError(t, err)

	for _, invoice := range []struct {
		id     string
		typeID string
		status string
		number string
	}{
		{id: gatheringInvoiceID, typeID: "gathering", status: "gathering", number: "GATHERING-1"},
		{id: standardInvoiceID, typeID: "standard", status: "draft.created", number: "INV-1"},
	} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO billing_invoices (
				id, namespace, created_at, updated_at,
				supplier_name, customer_name, number, type, status, currency,
				amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
				charges_total, discounts_total, credits_total, total,
				tax_app_id, invoicing_app_id, payment_app_id,
				source_billing_profile_id, workflow_config_id, customer_id
			) VALUES (
				$1, $2, NOW(), NOW(),
				'Tax migration supplier', 'Tax migration customer', $3, $4, $5, 'USD',
				0, 0, 0, 0, 0, 0, 0, 0,
				$6, $7, $8, $9, $10, $11
			)
		`, invoice.id, namespace, invoice.number, invoice.typeID, invoice.status,
			taxAppID, invoicingAppID, paymentAppID, profileID, invoiceWorkflowID, customerID)
		require.NoError(t, err)
	}

	return gatheringInvoiceID, standardInvoiceID
}

func seedGatheringInvoiceLineTaxCode(t *testing.T, db *sql.DB, namespace, id, key, stripeCode string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO tax_codes (
			id, namespace, created_at, updated_at, name, key, app_mappings
		) VALUES (
			$1, $2, NOW(), NOW(), $3, $4,
			jsonb_build_array(jsonb_build_object('app_type', 'stripe', 'tax_code', $5::text))
		)
	`, id, namespace, key, key, stripeCode)
	require.NoError(t, err)
}

func seedGatheringInvoiceLine(t *testing.T, db *sql.DB, namespace, invoiceID, id string, taxCodeID, taxBehavior, taxConfig any) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO billing_invoice_lines (
			id, namespace, created_at, updated_at,
			name, currency, type, status,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total,
			period_start, period_end, managed_by, invoice_at,
			tax_code_id, tax_behavior, tax_config, invoice_id
		) VALUES (
			$1, $2, NOW(), NOW(),
			'Tax migration line', 'USD', 'flat_fee', 'valid',
			0, 0, 0, 0, 0, 0, 0, 0,
			'2024-01-01 00:00:00', '2024-02-01 00:00:00', 'manual', '2024-02-01 00:00:00',
			$3, $4, $5::jsonb, $6
		)
	`, id, namespace, taxCodeID, taxBehavior, taxConfig, invoiceID)
	require.NoError(t, err)
}

func gatheringInvoiceLineTaxCodeIDByKey(t *testing.T, db *sql.DB, namespace, key string) string {
	t.Helper()

	var id string
	err := db.QueryRowContext(t.Context(), `
		SELECT id
		FROM tax_codes
		WHERE namespace = $1 AND key = $2 AND deleted_at IS NULL
	`, namespace, key).Scan(&id)
	require.NoError(t, err)

	return id
}

func assertGatheringInvoiceLineTax(t *testing.T, db *sql.DB, lineID, wantTaxCodeID, wantBehavior string) {
	t.Helper()

	var taxCodeID, embeddedTaxCodeID, behavior, embeddedBehavior sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT
			tax_code_id,
			tax_config ->> 'tax_code_id',
			tax_behavior,
			tax_config ->> 'behavior'
		FROM billing_invoice_lines
		WHERE id = $1
	`, lineID).Scan(&taxCodeID, &embeddedTaxCodeID, &behavior, &embeddedBehavior)
	require.NoError(t, err)

	if wantTaxCodeID == "" {
		require.False(t, taxCodeID.Valid)
		require.False(t, embeddedTaxCodeID.Valid)
	} else {
		require.Equal(t, wantTaxCodeID, taxCodeID.String)
		require.Equal(t, wantTaxCodeID, embeddedTaxCodeID.String)
	}

	if wantBehavior == "" {
		require.False(t, behavior.Valid)
		require.False(t, embeddedBehavior.Valid)
	} else {
		require.Equal(t, wantBehavior, behavior.String)
		require.Equal(t, wantBehavior, embeddedBehavior.String)
	}
}
