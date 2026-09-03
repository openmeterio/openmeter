package migrate_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const rateCardFeatureReferenceBackfillAnnotation = "dbmigration:backfill_rate_card_feature_references"

type rateCardFeatureReferenceMigrationState struct {
	FeatureID   sql.NullString
	FeatureKey  sql.NullString
	Annotations map[string]any
	UpdatedAt   time.Time
}

func queryRateCardFeatureReferenceMigrationStates(
	t testing.TB,
	db *sql.DB,
	table string,
	namespace string,
) map[string]rateCardFeatureReferenceMigrationState {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), fmt.Sprintf(`
		SELECT id, feature_id, feature_key, annotations::text, updated_at
		FROM %s
		WHERE namespace = $1
	`, table), namespace)
	require.NoError(t, err)
	defer rows.Close()

	states := map[string]rateCardFeatureReferenceMigrationState{}
	for rows.Next() {
		var (
			id              string
			state           rateCardFeatureReferenceMigrationState
			annotationsJSON sql.NullString
		)

		require.NoError(t, rows.Scan(
			&id,
			&state.FeatureID,
			&state.FeatureKey,
			&annotationsJSON,
			&state.UpdatedAt,
		))
		if annotationsJSON.Valid && annotationsJSON.String != "null" {
			require.NoError(t, json.Unmarshal([]byte(annotationsJSON.String), &state.Annotations))
		}

		states[id] = state
	}
	require.NoError(t, rows.Err())

	return states
}

func requireRateCardFeatureReferenceBackfillAnnotation(
	t testing.TB,
	state rateCardFeatureReferenceMigrationState,
	rateCardID string,
	featureID string,
	featureKey string,
	field string,
	migrationTimestamp *string,
) {
	t.Helper()

	rawAnnotation, ok := state.Annotations[rateCardFeatureReferenceBackfillAnnotation]
	require.True(t, ok)

	annotation, ok := rawAnnotation.(map[string]any)
	require.True(t, ok)
	require.Equal(t, field, annotation["field"])
	require.Equal(t, rateCardID, annotation["rate_card_id"])
	require.Equal(t, featureID, annotation["feature_id"])
	require.Equal(t, featureKey, annotation["feature_key"])

	annotationTimestamp, ok := annotation["at"].(string)
	require.True(t, ok)
	_, err := time.Parse(time.RFC3339Nano, annotationTimestamp)
	require.NoError(t, err)

	if *migrationTimestamp == "" {
		*migrationTimestamp = annotationTimestamp
	} else {
		require.Equal(t, *migrationTimestamp, annotationTimestamp)
	}
}

func assertRateCardFeatureReference(
	t testing.TB,
	db *sql.DB,
	table string,
	rateCardID string,
	wantFeatureID string,
	wantFeatureKey string,
	wantUpdatedAt time.Time,
) {
	t.Helper()

	var featureID, featureKey sql.NullString
	var updatedAt time.Time

	err := db.QueryRowContext(
		t.Context(),
		fmt.Sprintf(`SELECT feature_id, feature_key, updated_at FROM %s WHERE id = $1`, table),
		rateCardID,
	).Scan(&featureID, &featureKey, &updatedAt)
	require.NoError(t, err)

	require.Equal(t, wantFeatureID, featureID.String)
	require.Equal(t, wantFeatureID != "", featureID.Valid)
	require.Equal(t, wantFeatureKey, featureKey.String)
	require.Equal(t, wantFeatureKey != "", featureKey.Valid)
	require.Equal(t, wantUpdatedAt, updatedAt.UTC(), "the backfill must preserve rate-card updated_at")
}

func TestBackfillRateCardFeatureReferencesMigration(t *testing.T) {
	t.Parallel()

	const (
		namespace       = "rate_card_feature_backfill"
		otherNamespace  = "rate_card_feature_backfill_other"
		previousVersion = uint(20260814113138)
		targetVersion   = uint(20260814113139)
	)

	// given:
	// - ID-only and key-only plan and add-on rate cards, including a soft-deleted row
	// - complete and featureless rows that must remain untouched
	// - unresolved, cross-namespace, deleted-feature, and ambiguous references that cannot be repaired safely
	// when:
	// - the best-effort feature-reference backfill is migrated up and then down
	// then:
	// - only uniquely resolved rows are marked and repaired, and rollback restores those rows
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
	featureCreatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	featureVersionBoundary := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	featureDeletedAt := featureVersionBoundary
	beforeVersionBoundary := featureVersionBoundary.Add(-time.Hour)
	afterVersionBoundary := featureVersionBoundary.Add(time.Hour)
	ambiguousReferenceAt := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	ambiguousTerminalAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	oldVersionedFeatureID := ulid.Make().String()
	newVersionedFeatureID := ulid.Make().String()
	planIDOnlyFeatureID := ulid.Make().String()
	addonIDOnlyFeatureID := ulid.Make().String()
	planCompleteFeatureID := ulid.Make().String()
	addonCompleteFeatureID := ulid.Make().String()
	ambiguousFeatureID1 := ulid.Make().String()
	ambiguousFeatureID2 := ulid.Make().String()
	crossNamespaceFeatureID := ulid.Make().String()
	deletedFeatureID := ulid.Make().String()

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (
			id, namespace, created_at, updated_at, name, key, archived_at
		) VALUES
			($1, $10, $11, $11, 'Versioned feature v1', 'versioned', $12),
			($2, $10, $12, $12, 'Versioned feature v2', 'versioned', NULL),
			($3, $10, $11, $11, 'Plan ID-only feature', 'plan_id_only', $12),
			($4, $10, $11, $11, 'Add-on ID-only feature', 'addon_id_only', $12),
			($5, $10, $11, $11, 'Plan complete feature', 'plan_complete', NULL),
			($6, $10, $11, $11, 'Add-on complete feature', 'addon_complete', NULL),
			($7, $10, $11, $11, 'Ambiguous feature v1', 'ambiguous', $13),
			($8, $10, $11, $11, 'Ambiguous feature v2', 'ambiguous', $13),
			($9, $14, $11, $11, 'Other namespace feature', 'cross_namespace', NULL)
	`,
		oldVersionedFeatureID,
		newVersionedFeatureID,
		planIDOnlyFeatureID,
		addonIDOnlyFeatureID,
		planCompleteFeatureID,
		addonCompleteFeatureID,
		ambiguousFeatureID1,
		ambiguousFeatureID2,
		crossNamespaceFeatureID,
		namespace,
		featureCreatedAt,
		featureVersionBoundary,
		ambiguousTerminalAt,
		otherNamespace,
	)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (
			id, namespace, created_at, updated_at, deleted_at, name, key
		) VALUES (
			$1, $2, $3, $4, $4, 'Deleted feature', 'deleted'
		)
	`, deletedFeatureID, namespace, featureCreatedAt, featureDeletedAt)
	require.NoError(t, err)

	planID := ulid.Make().String()
	planPhaseID := ulid.Make().String()
	addonID := ulid.Make().String()
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plans (
			id, namespace, created_at, updated_at, name, key, version,
			currency, billing_cadence, pro_rating_config
		) VALUES (
			$1, $2, $3, $3, 'Feature backfill plan', 'feature_backfill_plan', 1,
			'USD', 'P1M', '{"enabled":true,"mode":"prorate_prices"}'::jsonb
		)
	`, planID, namespace, featureCreatedAt)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plan_phases (
			id, namespace, created_at, updated_at, name, key, plan_id, index
		) VALUES (
			$1, $2, $3, $3, 'Feature backfill phase', 'feature_backfill_phase', $4, 0
		)
	`, planPhaseID, namespace, featureCreatedAt, planID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addons (
			id, namespace, created_at, updated_at, name, key, version, currency
		) VALUES (
			$1, $2, $3, $3, 'Feature backfill add-on', 'feature_backfill_addon', 1, 'USD'
		)
	`, addonID, namespace, featureCreatedAt)
	require.NoError(t, err)

	planIDOnlyRateCardID := ulid.Make().String()
	planKeyOnlyRateCardID := ulid.Make().String()
	planNeitherRateCardID := ulid.Make().String()
	planCompleteRateCardID := ulid.Make().String()
	planUnresolvedRateCardID := ulid.Make().String()
	planAmbiguousRateCardID := ulid.Make().String()
	planDeletedFeatureRateCardID := ulid.Make().String()
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plan_rate_cards (
			id, namespace, created_at, updated_at, deleted_at, name, key, type,
			phase_id, feature_id, feature_key
		) VALUES
			($1, $7, $8, $9, $9, 'Plan ID only', 'plan_id_only_rc', 'FLAT_FEE', $10, $11, NULL),
			($2, $7, $8, $12, NULL, 'Plan key only', 'plan_key_only_rc', 'FLAT_FEE', $10, NULL, 'versioned'),
			($3, $7, $8, $9, NULL, 'Plan neither', 'plan_neither_rc', 'FLAT_FEE', $10, NULL, NULL),
			($4, $7, $8, $9, NULL, 'Plan complete', 'plan_complete_rc', 'FLAT_FEE', $10, $13, 'plan_complete'),
			($5, $7, $8, $9, NULL, 'Plan unresolved', 'plan_unresolved_rc', 'FLAT_FEE', $10, NULL, 'missing'),
			($6, $7, $8, $14, NULL, 'Plan ambiguous', 'plan_ambiguous_rc', 'FLAT_FEE', $10, NULL, 'ambiguous')
	`,
		planIDOnlyRateCardID,
		planKeyOnlyRateCardID,
		planNeitherRateCardID,
		planCompleteRateCardID,
		planUnresolvedRateCardID,
		planAmbiguousRateCardID,
		namespace,
		featureCreatedAt,
		afterVersionBoundary,
		planPhaseID,
		planIDOnlyFeatureID,
		beforeVersionBoundary,
		planCompleteFeatureID,
		ambiguousReferenceAt,
	)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plan_rate_cards (
			id, namespace, created_at, updated_at, name, key, type,
			phase_id, feature_id, feature_key
		) VALUES (
			$1, $2, $3, $4, 'Plan deleted feature', 'plan_deleted_feature_rc',
			'FLAT_FEE', $5, NULL, 'deleted'
		)
	`, planDeletedFeatureRateCardID, namespace, featureCreatedAt, afterVersionBoundary, planPhaseID)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		UPDATE plan_rate_cards
		SET annotations = '{"existing":"preserved"}'::jsonb
		WHERE id = $1
	`, planKeyOnlyRateCardID)
	require.NoError(t, err)

	addonIDOnlyRateCardID := ulid.Make().String()
	addonKeyOnlyRateCardID := ulid.Make().String()
	addonNeitherRateCardID := ulid.Make().String()
	addonCompleteRateCardID := ulid.Make().String()
	addonCrossNamespaceRateCardID := ulid.Make().String()
	addonDeletedFeatureRateCardID := ulid.Make().String()
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addon_rate_cards (
			id, namespace, created_at, updated_at, name, key, type,
			addon_id, feature_id, feature_key
		) VALUES
			($1, $6, $7, $8, 'Add-on ID only', 'addon_id_only_rc', 'FLAT_FEE', $9, $10, NULL),
			($2, $6, $7, $11, 'Add-on key only', 'addon_key_only_rc', 'FLAT_FEE', $9, NULL, 'versioned'),
			($3, $6, $7, $8, 'Add-on neither', 'addon_neither_rc', 'FLAT_FEE', $9, NULL, NULL),
			($4, $6, $7, $8, 'Add-on complete', 'addon_complete_rc', 'FLAT_FEE', $9, $12, 'addon_complete'),
			($5, $6, $7, $8, 'Add-on cross namespace', 'addon_cross_namespace_rc', 'FLAT_FEE', $9, $13, NULL)
	`,
		addonIDOnlyRateCardID,
		addonKeyOnlyRateCardID,
		addonNeitherRateCardID,
		addonCompleteRateCardID,
		addonCrossNamespaceRateCardID,
		namespace,
		featureCreatedAt,
		afterVersionBoundary,
		addonID,
		addonIDOnlyFeatureID,
		featureVersionBoundary,
		addonCompleteFeatureID,
		crossNamespaceFeatureID,
	)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addon_rate_cards (
			id, namespace, created_at, updated_at, name, key, type,
			addon_id, feature_id, feature_key
		) VALUES (
			$1, $2, $3, $4, 'Add-on deleted feature', 'addon_deleted_feature_rc',
			'FLAT_FEE', $5, NULL, 'deleted'
		)
	`, addonDeletedFeatureRateCardID, namespace, featureCreatedAt, afterVersionBoundary, addonID)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(targetVersion))

	planStates := queryRateCardFeatureReferenceMigrationStates(t, db, "plan_rate_cards", namespace)
	addonStates := queryRateCardFeatureReferenceMigrationStates(t, db, "addon_rate_cards", namespace)
	var migrationTimestamp string

	require.Equal(t, planIDOnlyFeatureID, planStates[planIDOnlyRateCardID].FeatureID.String)
	require.Equal(t, "plan_id_only", planStates[planIDOnlyRateCardID].FeatureKey.String)
	require.Equal(t, afterVersionBoundary, planStates[planIDOnlyRateCardID].UpdatedAt.UTC())
	requireRateCardFeatureReferenceBackfillAnnotation(
		t,
		planStates[planIDOnlyRateCardID],
		planIDOnlyRateCardID,
		planIDOnlyFeatureID,
		"plan_id_only",
		"feature_key",
		&migrationTimestamp,
	)

	require.Equal(t, oldVersionedFeatureID, planStates[planKeyOnlyRateCardID].FeatureID.String)
	require.Equal(t, "versioned", planStates[planKeyOnlyRateCardID].FeatureKey.String)
	require.Equal(t, beforeVersionBoundary, planStates[planKeyOnlyRateCardID].UpdatedAt.UTC())
	require.Equal(t, "preserved", planStates[planKeyOnlyRateCardID].Annotations["existing"])
	requireRateCardFeatureReferenceBackfillAnnotation(
		t,
		planStates[planKeyOnlyRateCardID],
		planKeyOnlyRateCardID,
		oldVersionedFeatureID,
		"versioned",
		"feature_id",
		&migrationTimestamp,
	)

	require.Equal(t, addonIDOnlyFeatureID, addonStates[addonIDOnlyRateCardID].FeatureID.String)
	require.Equal(t, "addon_id_only", addonStates[addonIDOnlyRateCardID].FeatureKey.String)
	requireRateCardFeatureReferenceBackfillAnnotation(
		t,
		addonStates[addonIDOnlyRateCardID],
		addonIDOnlyRateCardID,
		addonIDOnlyFeatureID,
		"addon_id_only",
		"feature_key",
		&migrationTimestamp,
	)

	require.Equal(t, newVersionedFeatureID, addonStates[addonKeyOnlyRateCardID].FeatureID.String)
	require.Equal(t, "versioned", addonStates[addonKeyOnlyRateCardID].FeatureKey.String)
	requireRateCardFeatureReferenceBackfillAnnotation(
		t,
		addonStates[addonKeyOnlyRateCardID],
		addonKeyOnlyRateCardID,
		newVersionedFeatureID,
		"versioned",
		"feature_id",
		&migrationTimestamp,
	)

	for _, state := range []rateCardFeatureReferenceMigrationState{
		planStates[planNeitherRateCardID],
		planStates[planCompleteRateCardID],
		planStates[planUnresolvedRateCardID],
		planStates[planAmbiguousRateCardID],
		planStates[planDeletedFeatureRateCardID],
		addonStates[addonNeitherRateCardID],
		addonStates[addonCompleteRateCardID],
		addonStates[addonCrossNamespaceRateCardID],
		addonStates[addonDeletedFeatureRateCardID],
	} {
		require.NotContains(t, state.Annotations, rateCardFeatureReferenceBackfillAnnotation)
	}
	require.False(t, planStates[planUnresolvedRateCardID].FeatureID.Valid)
	require.False(t, planStates[planAmbiguousRateCardID].FeatureID.Valid)
	require.False(t, planStates[planDeletedFeatureRateCardID].FeatureID.Valid)
	require.False(t, addonStates[addonCrossNamespaceRateCardID].FeatureKey.Valid)
	require.False(t, addonStates[addonDeletedFeatureRateCardID].FeatureID.Valid)

	// A later edit can retain the marker while replacing the feature reference.
	// Rollback must not clear either part of the newer reference.
	_, err = db.ExecContext(t.Context(), `
		UPDATE plan_rate_cards
		SET feature_id = $2, feature_key = $3
		WHERE id = $1
	`, planIDOnlyRateCardID, planCompleteFeatureID, "plan_complete")
	require.NoError(t, err)

	// Product-catalog versioning recreates rate cards and can copy their annotations.
	// The copied marker does not own any field on the replacement row.
	clonedAddonID := ulid.Make().String()
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addons (
			id, namespace, created_at, updated_at, name, key, version, currency
		) VALUES (
			$1, $2, $3, $3, 'Cloned feature backfill add-on',
			'cloned_feature_backfill_addon', 1, 'USD'
		)
	`, clonedAddonID, namespace, featureCreatedAt)
	require.NoError(t, err)

	clonedAddonRateCardID := ulid.Make().String()
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addon_rate_cards (
			id, namespace, created_at, updated_at, name, key, type,
			addon_id, feature_id, feature_key, annotations
		)
		SELECT
			$1, namespace, created_at, updated_at, 'Cloned add-on rate card',
			'cloned_addon_rate_card', type, $3, feature_id, feature_key, annotations
		FROM addon_rate_cards
		WHERE id = $2
	`, clonedAddonRateCardID, addonKeyOnlyRateCardID, clonedAddonID)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(previousVersion))

	assertRateCardFeatureReference(
		t,
		db,
		"plan_rate_cards",
		planIDOnlyRateCardID,
		planCompleteFeatureID,
		"plan_complete",
		afterVersionBoundary,
	)
	assertRateCardFeatureReference(
		t,
		db,
		"plan_rate_cards",
		planKeyOnlyRateCardID,
		"",
		"versioned",
		beforeVersionBoundary,
	)
	assertRateCardFeatureReference(
		t,
		db,
		"addon_rate_cards",
		addonIDOnlyRateCardID,
		addonIDOnlyFeatureID,
		"",
		afterVersionBoundary,
	)
	assertRateCardFeatureReference(
		t,
		db,
		"addon_rate_cards",
		addonKeyOnlyRateCardID,
		"",
		"versioned",
		featureVersionBoundary,
	)
	assertRateCardFeatureReference(
		t,
		db,
		"addon_rate_cards",
		clonedAddonRateCardID,
		newVersionedFeatureID,
		"versioned",
		featureVersionBoundary,
	)

	planStates = queryRateCardFeatureReferenceMigrationStates(t, db, "plan_rate_cards", namespace)
	addonStates = queryRateCardFeatureReferenceMigrationStates(t, db, "addon_rate_cards", namespace)
	for _, state := range []rateCardFeatureReferenceMigrationState{
		planStates[planIDOnlyRateCardID],
		planStates[planKeyOnlyRateCardID],
		addonStates[addonIDOnlyRateCardID],
		addonStates[addonKeyOnlyRateCardID],
		addonStates[clonedAddonRateCardID],
	} {
		require.NotContains(t, state.Annotations, rateCardFeatureReferenceBackfillAnnotation)
	}
	require.Nil(t, planStates[planIDOnlyRateCardID].Annotations)
	require.Equal(t, "preserved", planStates[planKeyOnlyRateCardID].Annotations["existing"])
	require.Nil(t, addonStates[addonIDOnlyRateCardID].Annotations)
	require.Nil(t, addonStates[addonKeyOnlyRateCardID].Annotations)
	require.Nil(t, addonStates[clonedAddonRateCardID].Annotations)
}
