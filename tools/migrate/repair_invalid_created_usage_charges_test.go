package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	repairInvalidCreatedUsageChargesPreviousVersion = uint(20260911165152)
	repairInvalidCreatedUsageChargesTargetVersion   = uint(20260914072004)
)

type repairInvalidCreatedUsageChargeCase struct {
	name             string
	invoiceStatus    string
	managedBy        string
	hasRunHistory    bool
	wantDeleted      bool
	chargeID         string
	invoiceID        string
	lineID           string
	subscriptionID   string
	syncStateID      string
	workflowConfigID string
}

func TestRepairInvalidCreatedUsageChargesMigration(t *testing.T) {
	namespace := "default"
	now := time.Date(2026, 9, 14, 7, 30, 0, 0, time.UTC)

	cases := []repairInvalidCreatedUsageChargeCase{
		{
			name:          "invalid created line without realization",
			invoiceStatus: "draft.invalid_created",
			managedBy:     "subscription",
			wantDeleted:   true,
		},
		{
			name:          "charge with historical realization",
			invoiceStatus: "draft.invalid_created",
			managedBy:     "subscription",
			hasRunHistory: true,
		},
		{
			name:          "invoice outside invalid created",
			invoiceStatus: "draft.created",
			managedBy:     "subscription",
		},
		{
			name:          "manually managed charge",
			invoiceStatus: "draft.invalid_created",
			managedBy:     "manual",
		},
	}

	// given: the malformed row shape and the nearest structural exclusions
	// when: the repair migration runs
	// then: only the malformed subscription-owned line and charge are tombstoned
	runner{
		stops: stops{
			{
				version:   repairInvalidCreatedUsageChargesPreviousVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					base := newRepairInvalidCreatedUsageChargesBase()
					seedRepairInvalidCreatedUsageChargesBase(t, db, namespace, now, base)

					for idx := range cases {
						cases[idx].chargeID = ulid.Make().String()
						cases[idx].invoiceID = ulid.Make().String()
						cases[idx].lineID = ulid.Make().String()
						cases[idx].workflowConfigID = ulid.Make().String()
						if cases[idx].managedBy == "subscription" {
							cases[idx].subscriptionID = ulid.Make().String()
							cases[idx].syncStateID = ulid.Make().String()
						}

						seedRepairInvalidCreatedUsageChargeCase(t, db, namespace, now, base, cases[idx], idx)
					}
				},
			},
			{
				version:   repairInvalidCreatedUsageChargesTargetVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							assertRepairInvalidCreatedUsageChargeCase(t, db, tc)
						})
					}
				},
			},
		},
	}.Test(t)
}

type repairInvalidCreatedUsageChargesBase struct {
	customerID        string
	taxCodeID         string
	featureID         string
	profileID         string
	profileWorkflowID string
	taxAppID          string
	invoicingAppID    string
	paymentAppID      string
}

func newRepairInvalidCreatedUsageChargesBase() repairInvalidCreatedUsageChargesBase {
	return repairInvalidCreatedUsageChargesBase{
		customerID:        ulid.Make().String(),
		taxCodeID:         ulid.Make().String(),
		featureID:         ulid.Make().String(),
		profileID:         ulid.Make().String(),
		profileWorkflowID: ulid.Make().String(),
		taxAppID:          ulid.Make().String(),
		invoicingAppID:    ulid.Make().String(),
		paymentAppID:      ulid.Make().String(),
	}
}

func seedRepairInvalidCreatedUsageChargesBase(t *testing.T, db *sql.DB, namespace string, now time.Time, base repairInvalidCreatedUsageChargesBase) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO customers (id, namespace, metadata, created_at, updated_at, key, name, currency)
		VALUES ($1, $2, '{}'::jsonb, $3, $3, 'repair-customer', 'Repair customer', 'EUR')
	`, base.customerID, namespace, now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO tax_codes (id, namespace, metadata, created_at, updated_at, name, key)
		VALUES ($1, $2, '{}'::jsonb, $3, $3, 'Repair tax code', 'repair-tax-code')
	`, base.taxCodeID, namespace, now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (id, namespace, metadata, created_at, updated_at, name, key)
		VALUES ($1, $2, '{}'::jsonb, $3, $3, 'Repair feature', 'repair-feature')
	`, base.featureID, namespace, now)
	require.NoError(t, err)

	for _, app := range []struct {
		id      string
		appType string
		name    string
	}{
		{id: base.taxAppID, appType: "tax", name: "Tax app"},
		{id: base.invoicingAppID, appType: "invoicing", name: "Invoicing app"},
		{id: base.paymentAppID, appType: "payment", name: "Payment app"},
	} {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO apps (id, namespace, metadata, created_at, updated_at, name, description, type, status)
			VALUES ($1, $2, '{}'::jsonb, $3, $3, $4, '', $5, 'ready')
		`, app.id, namespace, now, app.name, app.appType)
		require.NoError(t, err)
	}

	seedRepairInvalidCreatedUsageChargesWorkflowConfig(t, db, namespace, now, base.profileWorkflowID)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_profiles (
			id, namespace, metadata, created_at, updated_at, name, tax_app_id, invoicing_app_id,
			payment_app_id, workflow_config_id, "default", supplier_name
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Repair profile', $4, $5,
			$6, $7, false, 'Supplier'
		)
	`, base.profileID, namespace, now, base.taxAppID, base.invoicingAppID, base.paymentAppID, base.profileWorkflowID)
	require.NoError(t, err)
}

func seedRepairInvalidCreatedUsageChargesWorkflowConfig(t *testing.T, db *sql.DB, namespace string, now time.Time, id string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO billing_workflow_configs (
			id, namespace, created_at, updated_at, collection_alignment, line_collection_period,
			invoice_auto_advance, invoice_draft_period, invoice_due_after, invoice_collection_method,
			invoice_progressive_billing, subscription_end_proration_mode, tax_enabled, tax_enforced
		) VALUES (
			$1, $2, $3, $3, 'subscription', 'P1D', true, 'P1D', 'P1D', 'charge_automatically',
			true, 'bill_actual_period', true, false
		)
	`, id, namespace, now)
	require.NoError(t, err)
}

func seedRepairInvalidCreatedUsageChargeCase(
	t *testing.T,
	db *sql.DB,
	namespace string,
	now time.Time,
	base repairInvalidCreatedUsageChargesBase,
	tc repairInvalidCreatedUsageChargeCase,
	idx int,
) {
	t.Helper()
	subscriptionID := sql.NullString{String: tc.subscriptionID, Valid: tc.subscriptionID != ""}

	if tc.subscriptionID != "" {
		_, err := db.ExecContext(t.Context(), `
			INSERT INTO subscriptions (
				id, namespace, created_at, updated_at, active_from, customer_id, currency,
				billing_anchor, billing_cadence, pro_rating_config
			) VALUES (
				$1, $2, $3, $3, $3, $4, 'EUR', $3, 'P1M',
				'{"enabled":true,"mode":"prorate_prices"}'::jsonb
			)
		`, tc.subscriptionID, namespace, now, base.customerID)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `
			INSERT INTO subscription_billing_sync_states (
				id, namespace, has_billables, synced_at, subscription_id
			) VALUES ($1, $2, true, $3, $4)
		`, tc.syncStateID, namespace, now, tc.subscriptionID)
		require.NoError(t, err)
	}

	seedRepairInvalidCreatedUsageChargesWorkflowConfig(t, db, namespace, now, tc.workflowConfigID)

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based (
			id, namespace, invoice_at, settlement_mode, feature_key, feature_id, rating_engine, price,
			service_period_from, service_period_to, billing_period_from, billing_period_to,
			full_service_period_from, full_service_period_to, unique_reference_id, currency,
			managed_by, subscription_id, annotations, metadata, created_at, updated_at, name, status,
			status_detailed, customer_id, tax_code_id
		) VALUES (
			$1, $2, $3, 'credit_then_invoice', 'repair-feature', $4, 'delta',
			'{"type":"unit","amount":"1"}'::jsonb,
			$3, $5, $3, $5, $3, $5, $6, 'EUR',
			$7, $8, '{}'::jsonb, '{}'::jsonb, $3, $3, $9, 'created',
			'created', $10, $11
		)
	`, tc.chargeID, namespace, now, base.featureID, now.Add(24*time.Hour), fmt.Sprintf("repair-charge-%d", idx), tc.managedBy, subscriptionID, "Charge "+tc.name, base.customerID, base.taxCodeID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charges (id, namespace, created_at, unique_reference_id, type, charge_usage_based_id)
		VALUES ($1, $2, $3, $4, 'usage_based', $1)
	`, tc.chargeID, namespace, now, fmt.Sprintf("repair-charge-%d", idx))
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_invoices (
			id, namespace, metadata, created_at, updated_at, supplier_name, customer_name, number,
			type, customer_id, source_billing_profile_id, currency, status, workflow_config_id,
			tax_app_id, invoicing_app_id, payment_app_id, amount, taxes_total, taxes_inclusive_total,
			taxes_exclusive_total, charges_total, discounts_total, credits_total, total
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Supplier', 'Repair customer', $4,
			'standard', $5, $6, 'EUR', $7, $8,
			$9, $10, $11, 0, 0, 0,
			0, 0, 0, 0, 0
		)
	`, tc.invoiceID, namespace, now, "INV-"+tc.invoiceID, base.customerID, base.profileID, tc.invoiceStatus, tc.workflowConfigID, base.taxAppID, base.invoicingAppID, base.paymentAppID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_invoice_lines (
			id, namespace, metadata, created_at, updated_at, name, period_start, period_end,
			invoice_at, type, status, currency, invoice_id, managed_by, amount, taxes_total,
			taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total,
			credits_total, total, charge_id, engine, tax_code_id, subscription_id
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, $4, $3, $5,
			$3, 'usage_based', 'valid', 'EUR', $6, $7, 0, 0,
			0, 0, 0, 0,
			0, 0, $8, 'charge_usagebased', $9, $10
		)
	`, tc.lineID, namespace, now, "Line "+tc.name, now.Add(24*time.Hour), tc.invoiceID, tc.managedBy, tc.chargeID, base.taxCodeID, subscriptionID)
	require.NoError(t, err)

	if tc.hasRunHistory {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO charge_usage_based_runs (
				id, namespace, created_at, updated_at, deleted_at, type, initial_type, stored_at_lt,
				service_period_to, detailed_lines_present, metered_quantity,
				no_fiat_transaction_required, charge_id, feature_id, amount, taxes_total,
				taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total,
				credits_total, total
			) VALUES (
				$1, $2, $3, $3, $3, 'partial_invoice', 'partial_invoice', $4,
				$4, false, 0,
				true, $5, $6, 0, 0,
				0, 0, 0, 0,
				0, 0
			)
		`, ulid.Make().String(), namespace, now, now.Add(time.Hour), tc.chargeID, base.featureID)
		require.NoError(t, err)
	}
}

func assertRepairInvalidCreatedUsageChargeCase(t *testing.T, db *sql.DB, tc repairInvalidCreatedUsageChargeCase) {
	t.Helper()

	var lineDeletedAt sql.NullTime
	err := db.QueryRowContext(t.Context(), `
		SELECT deleted_at
		FROM billing_invoice_lines
		WHERE id = $1
	`, tc.lineID).Scan(&lineDeletedAt)
	require.NoError(t, err)
	require.Equal(t, tc.wantDeleted, lineDeletedAt.Valid)

	var (
		chargeDeletedAt sql.NullTime
		intentDeletedAt sql.NullTime
		status          string
		statusDetailed  string
	)
	err = db.QueryRowContext(t.Context(), `
		SELECT deleted_at, intent_deleted_at, status, status_detailed
		FROM charge_usage_based
		WHERE id = $1
	`, tc.chargeID).Scan(&chargeDeletedAt, &intentDeletedAt, &status, &statusDetailed)
	require.NoError(t, err)
	require.Equal(t, tc.wantDeleted, chargeDeletedAt.Valid)
	require.Equal(t, tc.wantDeleted, intentDeletedAt.Valid)
	if tc.wantDeleted {
		require.Equal(t, "deleted", status)
		require.Equal(t, "deleted", statusDetailed)
	} else {
		require.Equal(t, "created", status)
		require.Equal(t, "created", statusDetailed)
	}

	var rootDeletedAt sql.NullTime
	err = db.QueryRowContext(t.Context(), `
		SELECT deleted_at
		FROM charges
		WHERE id = $1
	`, tc.chargeID).Scan(&rootDeletedAt)
	require.NoError(t, err)
	require.Equal(t, tc.wantDeleted, rootDeletedAt.Valid)

	var (
		invoiceDeletedAt sql.NullTime
		invoiceStatus    string
	)
	err = db.QueryRowContext(t.Context(), `
		SELECT deleted_at, status
		FROM billing_invoices
		WHERE id = $1
	`, tc.invoiceID).Scan(&invoiceDeletedAt, &invoiceStatus)
	require.NoError(t, err)
	require.False(t, invoiceDeletedAt.Valid)
	require.Equal(t, tc.invoiceStatus, invoiceStatus)

	if tc.subscriptionID != "" {
		var syncStateCount int
		err = db.QueryRowContext(t.Context(), `
			SELECT count(*)
			FROM subscription_billing_sync_states
			WHERE subscription_id = $1
		`, tc.subscriptionID).Scan(&syncStateCount)
		require.NoError(t, err)
		if tc.wantDeleted {
			require.Zero(t, syncStateCount)
		} else {
			require.Equal(t, 1, syncStateCount)
		}
	}

	var runCount int
	err = db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM charge_usage_based_runs
		WHERE charge_id = $1
	`, tc.chargeID).Scan(&runCount)
	require.NoError(t, err)
	if tc.hasRunHistory {
		require.Equal(t, 1, runCount)
	} else {
		require.Zero(t, runCount)
	}
}
