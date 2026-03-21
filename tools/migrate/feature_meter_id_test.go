package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestFeatureMeterIDMigration(t *testing.T) {
	activeMeterID := ulid.Make()
	deletedMeterOldID := ulid.Make()
	deletedMeterNewID := ulid.Make()

	activeMeterFeatureID := ulid.Make()
	deletedMeterFeatureID := ulid.Make()
	missingMeterFeatureID := ulid.Make()
	postMigrationFeatureID := ulid.Make()

	const (
		namespace        = "default"
		activeMeterKey   = "requests"
		deletedMeterKey  = "tokens"
		missingMeterKey  = "missing-meter"
		activeEventType  = "request.created"
		deletedEventType = "token.recorded"
	)

	oldDeletedAt := time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC)
	newDeletedAt := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)

	runner{
		stops: stops{
			{
				// Setup legacy feature rows before meter_id exists.
				// Before: 20260320084936_charges-state-details.up.sql
				// After: 20260320171954_feature-meter-id.up.sql
				version:   20260320084936,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`
						INSERT INTO meters (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							name,
							event_type,
							value_property,
							group_by,
							aggregation,
							event_from,
							deleted_at
						)
						VALUES
							($1, $2, NOW(), NOW(), $3, 'Active meter', $4, NULL, '{}'::jsonb, 'COUNT', NOW(), NULL),
							($1, $5, NOW(), NOW(), $6, 'Deleted meter old', $7, NULL, '{}'::jsonb, 'COUNT', NOW(), $8),
							($1, $9, NOW(), NOW(), $6, 'Deleted meter new', $7, NULL, '{}'::jsonb, 'COUNT', NOW(), $10)
					`,
						namespace,
						activeMeterID.String(), activeMeterKey, activeEventType,
						deletedMeterOldID.String(), deletedMeterKey, deletedEventType, oldDeletedAt,
						deletedMeterNewID.String(), newDeletedAt,
					)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_slug
						)
						VALUES
							($1, $2, NOW(), NOW(), 'active meter feature', 'active_meter_feature', $3),
							($1, $4, NOW(), NOW(), 'deleted meter feature', 'deleted_meter_feature', $5),
							($1, $6, NOW(), NOW(), 'missing meter feature', 'missing_meter_feature', $7)
					`,
						namespace,
						activeMeterFeatureID.String(), activeMeterKey,
						deletedMeterFeatureID.String(), deletedMeterKey,
						missingMeterFeatureID.String(), missingMeterKey,
					)
					require.NoError(t, err)
				},
			},
			{
				// Verify backfill results after meter_id is introduced.
				version:   20260320171954,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var meterID sql.NullString

					err := db.QueryRow(`SELECT meter_id FROM features WHERE id = $1`, activeMeterFeatureID.String()).Scan(&meterID)
					require.NoError(t, err)
					require.True(t, meterID.Valid)
					require.Equal(t, activeMeterID.String(), meterID.String, "legacy feature should bind to the active meter for a reused key")

					err = db.QueryRow(`SELECT meter_id FROM features WHERE id = $1`, deletedMeterFeatureID.String()).Scan(&meterID)
					require.NoError(t, err)
					require.True(t, meterID.Valid)
					require.Equal(t, deletedMeterNewID.String(), meterID.String, "legacy feature should bind to the most recently deleted meter when no active meter exists")

					err = db.QueryRow(`SELECT meter_id FROM features WHERE id = $1`, missingMeterFeatureID.String()).Scan(&meterID)
					require.NoError(t, err)
					require.False(t, meterID.Valid, "unmatched meter_slug should remain unmigrated")

					var columnExists bool
					err = db.QueryRow(`
						SELECT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'features'
							  AND column_name = 'meter_id'
						)
					`).Scan(&columnExists)
					require.NoError(t, err)
					require.True(t, columnExists)

					var indexExists bool
					err = db.QueryRow(`
						SELECT EXISTS (
							SELECT 1
							FROM pg_indexes
							WHERE tablename = 'features'
							  AND indexname = 'feature_namespace_meter_id'
						)
					`).Scan(&indexExists)
					require.NoError(t, err)
					require.True(t, indexExists)

					_, err = db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_slug,
							meter_id
						)
						VALUES ($1, $2, NOW(), NOW(), 'post migration feature', 'post_migration_feature', NULL, $3)
					`,
						namespace,
						postMigrationFeatureID.String(),
						activeMeterID.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// Verify rollback preserves meter_slug for rows created after the migration.
				version:   20260320084936,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					var columnExists bool
					err := db.QueryRow(`
						SELECT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'features'
							  AND column_name = 'meter_id'
						)
					`).Scan(&columnExists)
					require.NoError(t, err)
					require.False(t, columnExists)

					var meterSlug sql.NullString
					err = db.QueryRow(`SELECT meter_slug FROM features WHERE id = $1`, postMigrationFeatureID.String()).Scan(&meterSlug)
					require.NoError(t, err)
					require.True(t, meterSlug.Valid)
					require.Equal(t, activeMeterKey, meterSlug.String, "down migration should restore meter_slug from meter_id")

					err = db.QueryRow(`SELECT meter_slug FROM features WHERE id = $1`, activeMeterFeatureID.String()).Scan(&meterSlug)
					require.NoError(t, err)
					require.True(t, meterSlug.Valid)
					require.Equal(t, activeMeterKey, meterSlug.String)
				},
			},
		},
	}.Test(t)
}
