package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRepairUsageBasedDeletedInvoiceRunsMigration(t *testing.T) {
	up := readMigration(t, "20260709132435_repair_usagebased_deleted_invoice_runs.up.sql")
	require.Contains(t, up, "CREATE TEMPORARY TABLE repair_usagebased_deleted_invoice_runs")
	require.Contains(t, up, "BEGIN;")
	require.Contains(t, up, "COMMIT;")
	require.NotContains(t, up, "c.intent_deleted_at = now()")
	require.NotContains(t, up, "om_func_generate_ulid()")

	namespace := "default"
	now := time.Date(2026, 7, 3, 15, 40, 11, 0, time.UTC)
	deletedAt := now.Add(time.Hour)

	customerID := ulid.Make().String()
	taxCodeID := ulid.Make().String()
	featureID := ulid.Make().String()
	workflowID := ulid.Make().String()
	profileID := ulid.Make().String()
	taxAppID := ulid.Make().String()
	invoicingAppID := ulid.Make().String()
	paymentAppID := ulid.Make().String()
	gatheringInvoiceID := ulid.Make().String()
	gatheringWorkflowID := ulid.Make().String()
	emptyGatheringCustomerID := ulid.Make().String()
	emptyGatheringInvoiceID := ulid.Make().String()
	emptyGatheringWorkflowID := ulid.Make().String()

	cases := []repairUsageBasedDeletedInvoiceRunsCase{
		{
			name:              "partial zero credit line deleted",
			runType:           "partial_invoice",
			creditsTotal:      "0",
			invoiceStatus:     "paid",
			lineDeletedAt:     &deletedAt,
			wantRunDeleted:    true,
			wantChargeDeleted: false,
			wantOverride:      false,
		},
		{
			name:              "final zero credit invoice deleted",
			runType:           "final_realization",
			creditsTotal:      "0",
			invoiceStatus:     "deleted",
			invoiceDeletedAt:  &deletedAt,
			wantRunDeleted:    true,
			wantChargeDeleted: true,
			wantOverride:      true,
		},
		{
			name:                        "final zero credit invoice deleted deletes empty gathering invoice",
			customerID:                  emptyGatheringCustomerID,
			gatheringInvoiceID:          emptyGatheringInvoiceID,
			runType:                     "final_realization",
			creditsTotal:                "0",
			invoiceStatus:               "deleted",
			invoiceDeletedAt:            &deletedAt,
			wantRunDeleted:              true,
			wantChargeDeleted:           true,
			wantGatheringInvoiceDeleted: true,
			wantOverride:                true,
		},
		{
			name:                 "final zero credit existing override",
			runType:              "final_realization",
			creditsTotal:         "0",
			invoiceStatus:        "deleted",
			invoiceDeletedAt:     &deletedAt,
			existingOverrideName: "existing override",
			wantRunDeleted:       true,
			wantChargeDeleted:    true,
			wantOverride:         true,
		},
		{
			name:              "final nonzero credit ignored",
			runType:           "final_realization",
			creditsTotal:      "1",
			invoiceStatus:     "deleted",
			invoiceDeletedAt:  &deletedAt,
			wantRunDeleted:    false,
			wantChargeDeleted: false,
			wantOverride:      false,
		},
		{
			name:              "deleted gathering invoice ignored",
			runType:           "final_realization",
			creditsTotal:      "0",
			invoiceStatus:     "gathering",
			invoiceDeletedAt:  &deletedAt,
			wantRunDeleted:    false,
			wantChargeDeleted: false,
			wantOverride:      false,
		},
		{
			name:              "active invoice and line ignored",
			runType:           "partial_invoice",
			creditsTotal:      "0",
			invoiceStatus:     "paid",
			wantRunDeleted:    false,
			wantChargeDeleted: false,
			wantOverride:      false,
		},
	}

	runner{
		stops: stops{
			{
				version:   20260703084504,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedRepairUsageBasedDeletedInvoiceRunsBase(t, db, repairUsageBasedDeletedInvoiceRunsBase{
						namespace:                namespace,
						now:                      now,
						customerID:               customerID,
						taxCodeID:                taxCodeID,
						featureID:                featureID,
						workflowID:               workflowID,
						profileID:                profileID,
						taxAppID:                 taxAppID,
						invoicingAppID:           invoicingAppID,
						paymentAppID:             paymentAppID,
						gatheringInvoiceID:       gatheringInvoiceID,
						gatheringWorkflowID:      gatheringWorkflowID,
						emptyGatheringCustomerID: emptyGatheringCustomerID,
						emptyGatheringInvoiceID:  emptyGatheringInvoiceID,
						emptyGatheringWorkflowID: emptyGatheringWorkflowID,
					})

					for idx := range cases {
						if cases[idx].customerID == "" {
							cases[idx].customerID = customerID
						}
						if cases[idx].gatheringInvoiceID == "" {
							cases[idx].gatheringInvoiceID = gatheringInvoiceID
						}
						cases[idx].chargeID = ulid.Make().String()
						cases[idx].invoiceID = ulid.Make().String()
						cases[idx].invoiceWorkflowID = ulid.Make().String()
						cases[idx].lineID = ulid.Make().String()
						cases[idx].gatheringLineID = ulid.Make().String()
						cases[idx].runID = ulid.Make().String()

						seedRepairUsageBasedDeletedInvoiceRunCase(t, db, namespace, now, taxCodeID, featureID, profileID, taxAppID, invoicingAppID, paymentAppID, cases[idx])
					}
				},
			},
			{
				version:   20260709132435,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							assertRepairUsageBasedDeletedInvoiceRunCase(t, db, tc)
						})
					}
				},
			},
		},
	}.Test(t)
}

type repairUsageBasedDeletedInvoiceRunsBase struct {
	namespace                string
	now                      time.Time
	customerID               string
	taxCodeID                string
	featureID                string
	workflowID               string
	profileID                string
	taxAppID                 string
	invoicingAppID           string
	paymentAppID             string
	gatheringInvoiceID       string
	gatheringWorkflowID      string
	emptyGatheringCustomerID string
	emptyGatheringInvoiceID  string
	emptyGatheringWorkflowID string
}

type repairUsageBasedDeletedInvoiceRunsCase struct {
	name                        string
	customerID                  string
	chargeID                    string
	invoiceID                   string
	invoiceWorkflowID           string
	lineID                      string
	gatheringInvoiceID          string
	gatheringLineID             string
	runID                       string
	runType                     string
	creditsTotal                string
	invoiceStatus               string
	invoiceDeletedAt            *time.Time
	lineDeletedAt               *time.Time
	existingOverrideName        string
	wantRunDeleted              bool
	wantChargeDeleted           bool
	wantGatheringInvoiceDeleted bool
	wantOverride                bool
}

func seedRepairUsageBasedDeletedInvoiceRunsBase(t *testing.T, db *sql.DB, input repairUsageBasedDeletedInvoiceRunsBase) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO customers (
			id, namespace, metadata, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'customer', 'Customer', 'EUR'
		)
	`, input.customerID, input.namespace, input.now)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO customers (
			id, namespace, metadata, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'empty-gathering-customer', 'Empty gathering customer', 'EUR'
		)
	`, input.emptyGatheringCustomerID, input.namespace, input.now)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO tax_codes (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Tax code', 'tax-code'
		)
	`, input.taxCodeID, input.namespace, input.now)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO features (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Feature', 'feature'
		)
	`, input.featureID, input.namespace, input.now)
	require.NoError(t, err)

	for _, app := range []struct {
		id      string
		appType string
		name    string
	}{
		{id: input.taxAppID, appType: "tax", name: "Tax app"},
		{id: input.invoicingAppID, appType: "invoicing", name: "Invoicing app"},
		{id: input.paymentAppID, appType: "payment", name: "Payment app"},
	} {
		_, err = db.Exec(`
			INSERT INTO apps (
				id, namespace, metadata, created_at, updated_at, name, description, type, status
			) VALUES (
				$1, $2, '{}'::jsonb, $3, $3, $4, '', $5, 'ready'
			)
		`, app.id, input.namespace, input.now, app.name, app.appType)
		require.NoError(t, err)
	}

	seedRepairUsageBasedDeletedInvoiceRunsWorkflowConfig(t, db, input.namespace, input.now, input.workflowID)
	seedRepairUsageBasedDeletedInvoiceRunsWorkflowConfig(t, db, input.namespace, input.now, input.gatheringWorkflowID)
	seedRepairUsageBasedDeletedInvoiceRunsWorkflowConfig(t, db, input.namespace, input.now, input.emptyGatheringWorkflowID)

	_, err = db.Exec(`
		INSERT INTO billing_profiles (
			id, namespace, metadata, created_at, updated_at, name, tax_app_id, invoicing_app_id,
			payment_app_id, workflow_config_id, "default", supplier_name
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Profile', $4, $5, $6, $7, false, 'Supplier'
		)
	`, input.profileID, input.namespace, input.now, input.taxAppID, input.invoicingAppID, input.paymentAppID, input.workflowID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoices (
			id, namespace, metadata, created_at, updated_at, supplier_name,
			customer_name, number, type, customer_id, source_billing_profile_id, currency,
			status, workflow_config_id, tax_app_id, invoicing_app_id, payment_app_id,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total,
			discounts_total, credits_total, total
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Supplier',
			'Customer', $4, 'gathering', $5, $6, 'EUR',
			'gathering', $7, $8, $9, $10,
			0, 0, 0, 0, 0,
			0, 0, 0
		)
	`, input.gatheringInvoiceID, input.namespace, input.now, "GATHERING-"+input.gatheringInvoiceID, input.customerID, input.profileID, input.gatheringWorkflowID, input.taxAppID, input.invoicingAppID, input.paymentAppID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoices (
			id, namespace, metadata, created_at, updated_at, supplier_name,
			customer_name, number, type, customer_id, source_billing_profile_id, currency,
			status, workflow_config_id, tax_app_id, invoicing_app_id, payment_app_id,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total,
			discounts_total, credits_total, total
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Supplier',
			'Empty gathering customer', $4, 'gathering', $5, $6, 'EUR',
			'gathering', $7, $8, $9, $10,
			0, 0, 0, 0, 0,
			0, 0, 0
		)
	`, input.emptyGatheringInvoiceID, input.namespace, input.now, "GATHERING-"+input.emptyGatheringInvoiceID, input.emptyGatheringCustomerID, input.profileID, input.emptyGatheringWorkflowID, input.taxAppID, input.invoicingAppID, input.paymentAppID)
	require.NoError(t, err)
}

func seedRepairUsageBasedDeletedInvoiceRunsWorkflowConfig(t *testing.T, db *sql.DB, namespace string, now time.Time, workflowID string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO billing_workflow_configs (
			id, namespace, created_at, updated_at, collection_alignment, line_collection_period,
			invoice_auto_advance, invoice_draft_period, invoice_due_after, invoice_collection_method,
			invoice_progressive_billing, subscription_end_proration_mode, tax_enabled, tax_enforced
		) VALUES (
			$1, $2, $3, $3, 'subscription', 'P1D', true, 'P1D', 'P1D', 'charge_automatically',
			true, 'bill_actual_period', true, false
		)
	`, workflowID, namespace, now)
	require.NoError(t, err)
}

func seedRepairUsageBasedDeletedInvoiceRunCase(
	t *testing.T,
	db *sql.DB,
	namespace string,
	now time.Time,
	taxCodeID string,
	featureID string,
	profileID string,
	taxAppID string,
	invoicingAppID string,
	paymentAppID string,
	tc repairUsageBasedDeletedInvoiceRunsCase,
) {
	t.Helper()

	seedRepairUsageBasedDeletedInvoiceRunsWorkflowConfig(t, db, namespace, now, tc.invoiceWorkflowID)

	_, err := db.Exec(`
		INSERT INTO charge_usage_based (
			id, namespace, invoice_at, settlement_mode, discounts, feature_key, price,
			service_period_from, service_period_to, billing_period_from, billing_period_to,
			full_service_period_from, full_service_period_to, unique_reference_id, currency,
			managed_by, annotations, metadata, created_at, updated_at, name, status,
			status_detailed, customer_id, tax_code_id, feature_id, rating_engine,
			current_realization_run_id
		) VALUES (
			$1, $2, $3, 'credit_then_invoice', '{}'::jsonb, 'feature', '{"type":"unit","amount":"1"}'::jsonb,
			$3, $4, $3, $4, $3, $5, $6, 'EUR',
			'subscription', '{}'::jsonb, '{}'::jsonb, $3, $3, $7, 'active',
			'active', $8, $9, $10, 'delta',
			NULL
		)
	`, tc.chargeID, namespace, now, now.Add(time.Hour), now.Add(24*time.Hour), "repair-"+tc.chargeID, "Charge "+tc.name, tc.customerID, taxCodeID, featureID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO charges (
			id, namespace, created_at, type, charge_usage_based_id
		) VALUES (
			$1, $2, $3, 'usage_based', $1
		)
	`, tc.chargeID, namespace, now)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoices (
			id, namespace, metadata, created_at, updated_at, deleted_at, supplier_name,
			customer_name, number, type, customer_id, source_billing_profile_id, currency,
			status, workflow_config_id, tax_app_id, invoicing_app_id, payment_app_id,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total,
			discounts_total, credits_total, total
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, $4, 'Supplier',
			'Customer', $5, 'standard', $6, $7, 'EUR',
			$8, $9, $10, $11, $12,
			0, 0, 0, 0, 0,
			0, 0, 0
		)
	`, tc.invoiceID, namespace, now, tc.invoiceDeletedAt, "INV-"+tc.invoiceID, tc.customerID, profileID, tc.invoiceStatus, tc.invoiceWorkflowID, taxAppID, invoicingAppID, paymentAppID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoice_lines (
			id, namespace, metadata, created_at, updated_at, deleted_at, name,
			period_start, period_end, invoice_at, type, status, currency, invoice_id,
			managed_by, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total, charge_id, engine,
			tax_code_id
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, $4, $5,
			$3, $6, $3, 'usage_based', 'valid', 'EUR', $7,
			'subscription', 0, 0, 0, 0,
			0, 0, 0, 0, $8, 'charge_usagebased',
			$9
		)
	`, tc.lineID, namespace, now, tc.lineDeletedAt, "Line "+tc.name, now.Add(time.Hour), tc.invoiceID, tc.chargeID, taxCodeID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoice_lines (
			id, namespace, metadata, created_at, updated_at, name,
			period_start, period_end, invoice_at, type, status, currency, invoice_id,
			managed_by, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total, charge_id, engine,
			tax_code_id
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, $4,
			$3, $5, $3, 'usage_based', 'valid', 'EUR', $6,
			'subscription', 0, 0, 0, 0,
			0, 0, 0, 0, $7, 'charge_usagebased',
			$8
		)
	`, tc.gatheringLineID, namespace, now, "Gathering line "+tc.name, now.Add(24*time.Hour), tc.gatheringInvoiceID, tc.chargeID, taxCodeID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO charge_usage_based_runs (
			id, namespace, created_at, updated_at, type, initial_type, stored_at_lt,
			service_period_to, detailed_lines_present, line_id, invoice_id, metered_quantity,
			no_fiat_transaction_required, charge_id, feature_id, amount, taxes_total,
			taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total,
			credits_total, total
		) VALUES (
			$1, $2, $3, $3, $4, $4, $6,
			$6, false, $7, $8, 0,
			true, $9, $10, 0, 0,
			0, 0, 0, 0,
			$5, 0
		)
	`, tc.runID, namespace, now, tc.runType, tc.creditsTotal, now.Add(time.Hour), tc.lineID, tc.invoiceID, tc.chargeID, featureID)
	require.NoError(t, err)

	_, err = db.Exec(`
		UPDATE charge_usage_based
		SET current_realization_run_id = $1
		WHERE id = $2
	`, tc.runID, tc.chargeID)
	require.NoError(t, err)

	if tc.existingOverrideName != "" {
		_, err = db.Exec(`
			INSERT INTO charge_usage_based_overrides (
				id, namespace, charge_id, name, metadata, service_period_from, service_period_to,
				full_service_period_from, full_service_period_to, billing_period_from, billing_period_to,
				invoice_at, feature_key, price, discounts
			) VALUES (
				$1, $2, $3, $4, '{}'::jsonb, $5, $6,
				$5, $7, $5, $6,
				$5, 'feature', '{"type":"unit","amount":"2"}'::jsonb, '{}'::jsonb
			)
		`, ulid.Make().String(), namespace, tc.chargeID, tc.existingOverrideName, now, now.Add(time.Hour), now.Add(24*time.Hour))
		require.NoError(t, err)
	}
}

func assertRepairUsageBasedDeletedInvoiceRunCase(t *testing.T, db *sql.DB, tc repairUsageBasedDeletedInvoiceRunsCase) {
	t.Helper()

	var runDeletedAt sql.NullTime
	err := db.QueryRow(`
		SELECT deleted_at
		FROM charge_usage_based_runs
		WHERE id = $1
	`, tc.runID).Scan(&runDeletedAt)
	require.NoError(t, err)
	require.Equal(t, tc.wantRunDeleted, runDeletedAt.Valid)

	var chargeDeletedAt sql.NullTime
	var baseIntentDeletedAt sql.NullTime
	var currentRunID sql.NullString
	var chargeStatus string
	var chargeDetailedStatus string
	err = db.QueryRow(`
		SELECT deleted_at, intent_deleted_at, current_realization_run_id, status, status_detailed
		FROM charge_usage_based
		WHERE id = $1
	`, tc.chargeID).Scan(&chargeDeletedAt, &baseIntentDeletedAt, &currentRunID, &chargeStatus, &chargeDetailedStatus)
	require.NoError(t, err)
	require.Equal(t, tc.wantChargeDeleted, chargeDeletedAt.Valid)
	require.False(t, baseIntentDeletedAt.Valid, "base/system intent must stay visible for subscription sync")
	require.Equal(t, !tc.wantRunDeleted, currentRunID.Valid)
	if !tc.wantRunDeleted {
		require.Equal(t, tc.runID, currentRunID.String)
	}
	if tc.wantChargeDeleted {
		require.Equal(t, "deleted", chargeStatus)
		require.Equal(t, "deleted", chargeDetailedStatus)
	} else {
		require.Equal(t, "active", chargeStatus)
		require.Equal(t, "active", chargeDetailedStatus)
	}

	var gatheringLineDeletedAt sql.NullTime
	err = db.QueryRow(`
		SELECT deleted_at
		FROM billing_invoice_lines
		WHERE id = $1
	`, tc.gatheringLineID).Scan(&gatheringLineDeletedAt)
	require.NoError(t, err)
	require.Equal(t, tc.wantChargeDeleted, gatheringLineDeletedAt.Valid)

	var gatheringInvoiceDeletedAt sql.NullTime
	err = db.QueryRow(`
		SELECT deleted_at
		FROM billing_invoices
		WHERE id = $1
	`, tc.gatheringInvoiceID).Scan(&gatheringInvoiceDeletedAt)
	require.NoError(t, err)
	require.Equal(t, tc.wantGatheringInvoiceDeleted, gatheringInvoiceDeletedAt.Valid)

	var overrideName sql.NullString
	var overrideIntentDeletedAt sql.NullTime
	err = db.QueryRow(`
		SELECT name, intent_deleted_at
		FROM charge_usage_based_overrides
		WHERE charge_id = $1
	`, tc.chargeID).Scan(&overrideName, &overrideIntentDeletedAt)
	if !tc.wantOverride {
		require.ErrorIs(t, err, sql.ErrNoRows)
		return
	}

	require.NoError(t, err)
	require.True(t, overrideIntentDeletedAt.Valid)
	require.True(t, overrideName.Valid)
	if tc.existingOverrideName != "" {
		require.Equal(t, tc.existingOverrideName, overrideName.String)
	} else {
		require.Equal(t, "Charge "+tc.name, overrideName.String)
	}
}
