package migrate_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const usageBasedRunImmutableBackfillAnnotation = "dbmigration:backfill_usage_based_run_immutable"

type usageBasedRunImmutableBackfillCase struct {
	name                 string
	invoiceStatus        string
	issuedAt             bool
	sentToCustomer       bool
	hasInvoicedUsage     bool
	immutable            bool
	existingAnnotation   string
	wantImmutableAfterUp bool
	runID                string
	invoiceID            string
	workflowConfigID     string
}

type usageBasedRunImmutableBackfillState struct {
	Immutable        bool
	HasInvoicedUsage bool
	Annotations      map[string]any
}

func TestBackfillUsageBasedRunImmutableMigration(t *testing.T) {
	t.Parallel()

	const (
		namespace       = "default"
		previousVersion = uint(20260825091451)
		targetVersion   = uint(20260826113408)
	)

	// given:
	// - historical runs across completed, in-progress, failed, and deleted invoice states
	// - a sent invoice whose status alone is ambiguous, a run without invoiced usage, and an already immutable run
	// when:
	// - the immutable backfill is migrated up and then down
	// then:
	// - only runs with invoiced usage and completed issuance evidence are marked, and rollback reverts only those runs
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { testDB.Close(t) })

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		sourceErr, databaseErr := migrator.Close()
		require.NoError(t, errors.Join(sourceErr, databaseErr))
	})
	require.NoError(t, migrator.Migrate(previousVersion))

	now := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	base := usageBasedRunImmutableBackfillBase{
		namespace:         namespace,
		now:               now,
		customerID:        ulid.Make().String(),
		taxCodeID:         ulid.Make().String(),
		featureID:         ulid.Make().String(),
		profileID:         ulid.Make().String(),
		profileWorkflowID: ulid.Make().String(),
		taxAppID:          ulid.Make().String(),
		invoicingAppID:    ulid.Make().String(),
		paymentAppID:      ulid.Make().String(),
		chargeID:          ulid.Make().String(),
	}
	seedUsageBasedRunImmutableBackfillBase(t, testDB.PGDriver.DB(), base)

	cases := []usageBasedRunImmutableBackfillCase{
		{
			name:                 "issued invoice",
			invoiceStatus:        "issued",
			hasInvoicedUsage:     true,
			existingAnnotation:   "preserved",
			wantImmutableAfterUp: true,
		},
		{
			name:                 "payment processing invoice",
			invoiceStatus:        "payment_processing.pending",
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:                 "overdue invoice",
			invoiceStatus:        "overdue",
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:                 "paid invoice",
			invoiceStatus:        "paid",
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:                 "uncollectible invoice",
			invoiceStatus:        "uncollectible",
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:                 "voided invoice",
			invoiceStatus:        "voided",
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:                 "sent invoice in failed issuing state",
			invoiceStatus:        "issuing.charge_booking_failed",
			sentToCustomer:       true,
			hasInvoicedUsage:     true,
			wantImmutableAfterUp: true,
		},
		{
			name:             "invoice still booking charges",
			invoiceStatus:    "issuing.charge_booking",
			hasInvoicedUsage: true,
		},
		{
			name:             "future issued at does not complete issuance",
			invoiceStatus:    "issuing.charge_booking_failed",
			issuedAt:         true,
			hasInvoicedUsage: true,
		},
		{
			name:             "deleted invoice without send evidence",
			invoiceStatus:    "deleted",
			hasInvoicedUsage: true,
		},
		{
			name:          "issued invoice without invoiced usage",
			invoiceStatus: "issued",
		},
		{
			name:                 "already immutable run",
			invoiceStatus:        "issued",
			hasInvoicedUsage:     true,
			immutable:            true,
			existingAnnotation:   "immutable",
			wantImmutableAfterUp: true,
		},
	}

	for idx := range cases {
		cases[idx].runID = ulid.Make().String()
		cases[idx].invoiceID = ulid.Make().String()
		cases[idx].workflowConfigID = ulid.Make().String()
		seedUsageBasedRunImmutableBackfillCase(t, testDB.PGDriver.DB(), base, cases[idx], idx)
	}

	require.NoError(t, migrator.Migrate(targetVersion))

	states := queryUsageBasedRunImmutableBackfillStates(t, testDB.PGDriver.DB(), base.chargeID)
	for _, tc := range cases {
		t.Run(tc.name+" after up", func(t *testing.T) {
			state, ok := states[tc.runID]
			require.True(t, ok)
			require.Equal(t, tc.hasInvoicedUsage, state.HasInvoicedUsage)
			require.Equal(t, tc.wantImmutableAfterUp, state.Immutable)

			_, marked := state.Annotations[usageBasedRunImmutableBackfillAnnotation]
			wantMarked := tc.wantImmutableAfterUp && !tc.immutable
			require.Equal(t, wantMarked, marked)
			if marked {
				migrationTimestamp, ok := state.Annotations[usageBasedRunImmutableBackfillAnnotation].(string)
				require.True(t, ok)
				_, err := time.Parse(time.RFC3339Nano, migrationTimestamp)
				require.NoError(t, err)
			}

			if tc.existingAnnotation != "" {
				require.Equal(t, tc.existingAnnotation, state.Annotations["existing"])
			}
		})
	}

	require.NoError(t, migrator.Migrate(previousVersion))

	states = queryUsageBasedRunImmutableBackfillStates(t, testDB.PGDriver.DB(), base.chargeID)
	for _, tc := range cases {
		t.Run(tc.name+" after down", func(t *testing.T) {
			state, ok := states[tc.runID]
			require.True(t, ok)
			require.Equal(t, tc.immutable, state.Immutable)
			require.NotContains(t, state.Annotations, usageBasedRunImmutableBackfillAnnotation)

			if tc.existingAnnotation == "" {
				require.Nil(t, state.Annotations)
			} else {
				require.Equal(t, tc.existingAnnotation, state.Annotations["existing"])
			}
		})
	}
}

type usageBasedRunImmutableBackfillBase struct {
	namespace         string
	now               time.Time
	customerID        string
	taxCodeID         string
	featureID         string
	profileID         string
	profileWorkflowID string
	taxAppID          string
	invoicingAppID    string
	paymentAppID      string
	chargeID          string
}

func seedUsageBasedRunImmutableBackfillBase(t *testing.T, db *sql.DB, input usageBasedRunImmutableBackfillBase) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO customers (
			id, namespace, metadata, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'immutable-backfill-customer', 'Customer', 'EUR'
		)
	`, input.customerID, input.namespace, input.now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO tax_codes (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Tax code', 'immutable-backfill-tax-code'
		)
	`, input.taxCodeID, input.namespace, input.now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Feature', 'immutable-backfill-feature'
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
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO apps (
				id, namespace, metadata, created_at, updated_at, name, description, type, status
			) VALUES (
				$1, $2, '{}'::jsonb, $3, $3, $4, '', $5, 'ready'
			)
		`, app.id, input.namespace, input.now, app.name, app.appType)
		require.NoError(t, err)
	}

	seedUsageBasedRunImmutableBackfillWorkflowConfig(t, db, input.namespace, input.now, input.profileWorkflowID)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO billing_profiles (
			id, namespace, metadata, created_at, updated_at, name, tax_app_id, invoicing_app_id,
			payment_app_id, workflow_config_id, "default", supplier_name
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Profile', $4, $5,
			$6, $7, false, 'Supplier'
		)
	`, input.profileID, input.namespace, input.now, input.taxAppID, input.invoicingAppID, input.paymentAppID, input.profileWorkflowID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based (
			id, namespace, invoice_at, settlement_mode, discounts, feature_key, price,
			service_period_from, service_period_to, billing_period_from, billing_period_to,
			full_service_period_from, full_service_period_to, unique_reference_id, currency,
			managed_by, annotations, metadata, created_at, updated_at, name, status,
			status_detailed, customer_id, tax_code_id, feature_id, rating_engine,
			current_realization_run_id
		) VALUES (
			$1, $2, $3, 'credit_then_invoice', '{}'::jsonb, 'immutable-backfill-feature',
			'{"type":"unit","amount":"1"}'::jsonb,
			$3, $4, $3, $4,
			$3, $4, 'immutable-backfill-charge', 'EUR',
			'subscription', '{}'::jsonb, '{}'::jsonb, $3, $3, 'Charge', 'active',
			'active', $5, $6, $7, 'delta',
			NULL
		)
	`, input.chargeID, input.namespace, input.now, input.now.Add(24*time.Hour), input.customerID, input.taxCodeID, input.featureID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charges (
			id, namespace, created_at, type, charge_usage_based_id
		) VALUES (
			$1, $2, $3, 'usage_based', $1
		)
	`, input.chargeID, input.namespace, input.now)
	require.NoError(t, err)
}

func seedUsageBasedRunImmutableBackfillWorkflowConfig(t *testing.T, db *sql.DB, namespace string, now time.Time, id string) {
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

func seedUsageBasedRunImmutableBackfillCase(
	t *testing.T,
	db *sql.DB,
	base usageBasedRunImmutableBackfillBase,
	tc usageBasedRunImmutableBackfillCase,
	caseIndex int,
) {
	t.Helper()

	seedUsageBasedRunImmutableBackfillWorkflowConfig(t, db, base.namespace, base.now, tc.workflowConfigID)

	var sentToCustomerAt *time.Time
	if tc.sentToCustomer {
		sentToCustomerAt = &base.now
	}
	var issuedAt *time.Time
	if tc.issuedAt {
		value := base.now.Add(time.Hour)
		issuedAt = &value
	}

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO billing_invoices (
			id, namespace, metadata, created_at, updated_at, supplier_name,
			customer_name, number, type, customer_id, source_billing_profile_id, currency,
			status, issued_at, sent_to_customer_at, workflow_config_id, tax_app_id, invoicing_app_id, payment_app_id,
			amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total, charges_total,
			discounts_total, credits_total, total
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Supplier',
			'Customer', $4, 'standard', $5, $6, 'EUR',
			$7, $8, $9, $10, $11, $12, $13,
			0, 0, 0, 0, 0,
			0, 0, 0
		)
	`,
		tc.invoiceID,
		base.namespace,
		base.now,
		"INV-IMMUTABLE-BACKFILL-"+tc.invoiceID,
		base.customerID,
		base.profileID,
		tc.invoiceStatus,
		issuedAt,
		sentToCustomerAt,
		tc.workflowConfigID,
		base.taxAppID,
		base.invoicingAppID,
		base.paymentAppID,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based_runs (
			id, namespace, created_at, updated_at, type, initial_type, stored_at_lt,
			service_period_to, detailed_lines_present, invoice_id, metered_quantity,
			no_fiat_transaction_required, immutable, charge_id, feature_id, amount, taxes_total,
			taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total,
			credits_total, total
		) VALUES (
			$1, $2, $3, $3, 'final_realization', 'final_realization', $4,
			$4, false, $5, 0,
			$6, $7, $8, $9, 0, 0,
			0, 0, 0, 0,
			0, 0
		)
	`,
		tc.runID,
		base.namespace,
		base.now.Add(time.Duration(caseIndex)*time.Second),
		base.now.Add(24*time.Hour),
		tc.invoiceID,
		!tc.hasInvoicedUsage,
		tc.immutable,
		base.chargeID,
		base.featureID,
	)
	require.NoError(t, err)

	if !tc.hasInvoicedUsage {
		return
	}

	var annotations any
	if tc.existingAnnotation != "" {
		annotationsJSON, err := json.Marshal(map[string]string{
			"existing": tc.existingAnnotation,
		})
		require.NoError(t, err)
		annotations = string(annotationsJSON)
	}

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based_run_invoiced_usages (
			id, namespace, created_at, updated_at, service_period_from, service_period_to,
			annotations, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total, run_id
		) VALUES (
			$1, $2, $3, $3, $3, $4,
			$5::jsonb, 0, 0, 0, 0,
			0, 0, 0, 0, $6
		)
	`, ulid.Make().String(), base.namespace, base.now, base.now.Add(24*time.Hour), annotations, tc.runID)
	require.NoError(t, err)
}

func queryUsageBasedRunImmutableBackfillStates(
	t testing.TB,
	db *sql.DB,
	chargeID string,
) map[string]usageBasedRunImmutableBackfillState {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT run.id, run.immutable, usage.run_id, usage.annotations::text
		FROM charge_usage_based_runs AS run
		LEFT JOIN charge_usage_based_run_invoiced_usages AS usage
			ON usage.run_id = run.id
		 AND usage.namespace = run.namespace
		WHERE run.charge_id = $1
	`, chargeID)
	require.NoError(t, err)
	defer rows.Close()

	states := map[string]usageBasedRunImmutableBackfillState{}
	for rows.Next() {
		var (
			runID           string
			usageRunID      sql.NullString
			annotationsJSON sql.NullString
			state           usageBasedRunImmutableBackfillState
		)

		require.NoError(t, rows.Scan(&runID, &state.Immutable, &usageRunID, &annotationsJSON))
		state.HasInvoicedUsage = usageRunID.Valid
		if annotationsJSON.Valid && annotationsJSON.String != "null" {
			require.NoError(t, json.Unmarshal([]byte(annotationsJSON.String), &state.Annotations))
		}

		states[runID] = state
	}
	require.NoError(t, rows.Err())

	return states
}
