package migrate_test

import (
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type usageBasedRunPriorLineageState struct {
	SchemaLevel int
	PriorRunID  sql.NullString
}

func queryUsageBasedRunPriorLineageStates(t testing.TB, db *sql.DB) map[string]usageBasedRunPriorLineageState {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT id, schema_level, prior_run_id
		FROM charge_usage_based_runs
	`)
	require.NoError(t, err)
	defer rows.Close()

	states := map[string]usageBasedRunPriorLineageState{}
	for rows.Next() {
		var (
			id    string
			state usageBasedRunPriorLineageState
		)

		require.NoError(t, rows.Scan(&id, &state.SchemaLevel, &state.PriorRunID))
		states[id] = state
	}
	require.NoError(t, rows.Err())

	return states
}

func TestBackfillUsageBasedRunPriorLineageMigration(t *testing.T) {
	t.Parallel()

	const (
		namespace       = "default"
		previousVersion = uint(20260826113408)
		targetVersion   = uint(20260826120555)
	)

	// given:
	// - legacy runs from two charges with interleaved creation times
	// - same-created-at runs inserted out of ID order, including voided history
	// - an existing schema-level-2 run
	// when:
	// - the lineage backfill is migrated up and then down
	// then:
	// - only legacy runs are linked by creation time and ID per charge
	// - all legacy runs are promoted, existing level-2 lineage is preserved, and rollback is non-destructive
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

	db := testDB.PGDriver.DB()
	now := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	customerID := ulid.Make().String()
	taxCodeID := ulid.Make().String()
	featureID := ulid.Make().String()
	chargeAID := ulid.Make().String()
	chargeBID := ulid.Make().String()

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO customers (
			id, namespace, metadata, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'lineage-customer', 'Lineage Customer', 'EUR'
		)
	`, customerID, namespace, now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO tax_codes (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Lineage tax code', 'lineage-tax-code'
		)
	`, taxCodeID, namespace, now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (
			id, namespace, metadata, created_at, updated_at, name, key
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, 'Lineage feature', 'lineage-feature'
		)
	`, featureID, namespace, now)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based (
			id, namespace, invoice_at, settlement_mode, discounts, feature_key, price,
			service_period_from, service_period_to, billing_period_from, billing_period_to,
			full_service_period_from, full_service_period_to, unique_reference_id, currency,
			managed_by, annotations, metadata, created_at, updated_at, name, status,
			status_detailed, customer_id, tax_code_id, feature_id, rating_engine,
			current_realization_run_id
		) VALUES
			(
				$1, $3, $4, 'credit_then_invoice', '{}'::jsonb, 'lineage-feature',
				'{"type":"unit","amount":"1"}'::jsonb,
				$4, $5, $4, $5, $4, $5, 'lineage-charge-a', 'EUR',
				'subscription', '{}'::jsonb, '{}'::jsonb, $4, $4, 'Lineage Charge A', 'active',
				'active', $6, $7, $8, 'delta', NULL
			),
			(
				$2, $3, $4, 'credit_then_invoice', '{}'::jsonb, 'lineage-feature',
				'{"type":"unit","amount":"1"}'::jsonb,
				$4, $5, $4, $5, $4, $5, 'lineage-charge-b', 'EUR',
				'subscription', '{}'::jsonb, '{}'::jsonb, $4, $4, 'Lineage Charge B', 'active',
				'active', $6, $7, $8, 'delta', NULL
			)
	`, chargeAID, chargeBID, namespace, now, now.Add(24*time.Hour), customerID, taxCodeID, featureID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charges (id, namespace, created_at, type, charge_usage_based_id)
		VALUES
			($1, $3, $4, 'usage_based', $1),
			($2, $3, $4, 'usage_based', $2)
	`, chargeAID, chargeBID, namespace, now)
	require.NoError(t, err)

	tiedRunIDs := []string{ulid.Make().String(), ulid.Make().String(), ulid.Make().String()}
	slices.Sort(tiedRunIDs)
	chargeAFirstRunID := tiedRunIDs[0]
	chargeAVoidedRunID := tiedRunIDs[1]
	chargeAThirdRunID := tiedRunIDs[2]
	chargeAFourthRunID := ulid.Make().String()
	chargeALevel2RunID := ulid.Make().String()
	chargeBFirstRunID := ulid.Make().String()
	chargeBSecondRunID := ulid.Make().String()

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO charge_usage_based_runs (
			id, namespace, created_at, updated_at, deleted_at, type, initial_type,
			stored_at_lt, service_period_to, detailed_lines_present, metered_quantity,
			no_fiat_transaction_required, charge_id, feature_id, amount, taxes_total,
			taxes_inclusive_total, taxes_exclusive_total, charges_total, discounts_total,
			credits_total, total, schema_level, prior_run_id
		) VALUES
			($3, $1, $10, $10, NULL, 'partial_invoice', 'partial_invoice', $10, $11, false, 1, true, $2, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL),
			($5, $1, $10, $10, NULL, 'partial_invoice', 'partial_invoice', $10, $11, false, 3, true, $2, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL),
			($4, $1, $10, $10, NULL, 'invalid_due_to_unsupported_credit_note', 'partial_invoice', $10, $11, false, 2, true, $2, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL),
			($6, $1, $12, $12, NULL, 'partial_invoice', 'partial_invoice', $12, $13, false, 4, true, $2, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL),
			($7, $1, $14, $14, NULL, 'final_realization', 'final_realization', $14, $15, false, 5, true, $2, $9, 0, 0, 0, 0, 0, 0, 0, 0, 2, $6),
			($16, $1, $17, $17, NULL, 'partial_invoice', 'partial_invoice', $17, $18, false, 1, true, $8, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL),
			($19, $1, $20, $20, NULL, 'final_realization', 'final_realization', $20, $21, false, 2, true, $8, $9, 0, 0, 0, 0, 0, 0, 0, 0, 1, NULL)
	`,
		namespace,
		chargeAID,
		chargeAFirstRunID,
		chargeAVoidedRunID,
		chargeAThirdRunID,
		chargeAFourthRunID,
		chargeALevel2RunID,
		chargeBID,
		featureID,
		now,
		now.Add(time.Hour),
		now.Add(2*time.Hour),
		now.Add(3*time.Hour),
		now.Add(4*time.Hour),
		now.Add(5*time.Hour),
		chargeBFirstRunID,
		now.Add(30*time.Minute),
		now.Add(90*time.Minute),
		chargeBSecondRunID,
		now.Add(3*time.Hour),
		now.Add(4*time.Hour),
	)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(targetVersion))

	wantStates := map[string]usageBasedRunPriorLineageState{
		chargeAFirstRunID:  {SchemaLevel: 2},
		chargeAVoidedRunID: {SchemaLevel: 2, PriorRunID: sql.NullString{String: chargeAFirstRunID, Valid: true}},
		chargeAThirdRunID:  {SchemaLevel: 2, PriorRunID: sql.NullString{String: chargeAVoidedRunID, Valid: true}},
		chargeAFourthRunID: {SchemaLevel: 2, PriorRunID: sql.NullString{String: chargeAThirdRunID, Valid: true}},
		chargeALevel2RunID: {SchemaLevel: 2, PriorRunID: sql.NullString{String: chargeAFourthRunID, Valid: true}},
		chargeBFirstRunID:  {SchemaLevel: 2},
		chargeBSecondRunID: {SchemaLevel: 2, PriorRunID: sql.NullString{String: chargeBFirstRunID, Valid: true}},
	}
	require.Equal(t, wantStates, queryUsageBasedRunPriorLineageStates(t, db))

	require.NoError(t, migrator.Migrate(previousVersion))
	require.Equal(t, wantStates, queryUsageBasedRunPriorLineageStates(t, db))
}
