package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestFeatureAdvancedMeterGroupByFiltersMigration(t *testing.T) {
	// Feature with null filters
	nullFeatureId := ulid.Make()

	// Feature with empty filters
	emptyFeatureId := ulid.Make()

	// Feature with single filter
	singleFeatureId := ulid.Make()

	// Feature with multiple filters
	multipleFeatureId := ulid.Make()

	runner{
		stops: stops{
			{
				// TEST SCENARIO 1: Setup test data before the migration
				// Creates four features with different meter_group_by_filters scenarios:
				// - Feature 1: NULL meter_group_by_filters
				//   Expected: advanced_meter_group_by_filters should remain NULL
				// - Feature 2: Empty meter_group_by_filters ({})
				//   Expected: advanced_meter_group_by_filters should be empty object ({})
				// - Feature 3: Single filter ({"region":"us-east-1"})
				//   Expected: advanced_meter_group_by_filters should be {"region":{"$eq":"us-east-1"}}
				// - Feature 4: Multiple filters ({"region":"us-east-1","environment":"production","team":"backend"})
				//   Expected: advanced_meter_group_by_filters should wrap each value with $eq operator
				//
				// Before: 20251006141236_feature-advanced-meter-group-by-filters.up.sql
				version:   20250926145930,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Create feature with null filters
					_, err := db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_group_by_filters
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'null_filters_feature',
							'null_filters_key',
							NULL
						)`,
						nullFeatureId.String(),
					)
					require.NoError(t, err)

					// Create feature with empty filters
					_, err = db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_group_by_filters
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'empty_filters_feature',
							'empty_filters_key',
							'{}'::jsonb
						)`,
						emptyFeatureId.String(),
					)
					require.NoError(t, err)

					// Create feature with single filter
					_, err = db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_group_by_filters
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'single_filter_feature',
							'single_filter_key',
							'{"region":"us-east-1"}'::jsonb
						)`,
						singleFeatureId.String(),
					)
					require.NoError(t, err)

					// Create feature with multiple filters
					_, err = db.Exec(`
						INSERT INTO features (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							meter_group_by_filters
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'multiple_filters_feature',
							'multiple_filters_key',
							'{"region":"us-east-1","environment":"production","team":"backend"}'::jsonb
						)`,
						multipleFeatureId.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// TEST SCENARIO 2: Verify migration results
				// This test verifies that the migration correctly:
				// 1. Adds the new advanced_meter_group_by_filters column
				// 2. Copies existing meter_group_by_filters data to advanced_meter_group_by_filters
				// 3. Transforms simple key-value pairs to FilterString objects with $eq operator
				// 4. Leaves original meter_group_by_filters column unchanged
				// 5. Handles NULL values correctly (doesn't copy NULL values)
				//
				// After: 20251006141236_feature-advanced-meter-group-by-filters.up.sql
				version:   20251006141236,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Check null filters feature
					var originalFilters, advancedFilters sql.NullString
					err := db.QueryRow(`
						SELECT meter_group_by_filters, advanced_meter_group_by_filters
						FROM features
						WHERE id = $1
					`, nullFeatureId.String()).Scan(&originalFilters, &advancedFilters)
					require.NoError(t, err)

					require.False(t, originalFilters.Valid, "Original filters should remain NULL")
					require.False(t, advancedFilters.Valid, "Advanced filters should be NULL when original is NULL")

					// Check empty filters feature
					err = db.QueryRow(`
						SELECT meter_group_by_filters, advanced_meter_group_by_filters
						FROM features
						WHERE id = $1
					`, emptyFeatureId.String()).Scan(&originalFilters, &advancedFilters)
					require.NoError(t, err)

					require.True(t, originalFilters.Valid, "Original filters should not be NULL")
					require.Equal(t, "{}", originalFilters.String, "Original filters should remain empty object")
					require.True(t, advancedFilters.Valid, "Advanced filters should not be NULL")
					require.Equal(t, "{}", advancedFilters.String, "Advanced filters should be empty object")

					// Check single filter feature
					err = db.QueryRow(`
						SELECT meter_group_by_filters, advanced_meter_group_by_filters
						FROM features
						WHERE id = $1
					`, singleFeatureId.String()).Scan(&originalFilters, &advancedFilters)
					require.NoError(t, err)

					require.True(t, originalFilters.Valid, "Original filters should not be NULL")
					require.JSONEq(t, `{"region":"us-east-1"}`, originalFilters.String, "Original filters should remain unchanged")
					require.True(t, advancedFilters.Valid, "Advanced filters should not be NULL")
					require.JSONEq(t, `{"region":{"$eq":"us-east-1"}}`, advancedFilters.String, "Advanced filters should have $eq wrapper")

					// Check multiple filters feature
					err = db.QueryRow(`
						SELECT meter_group_by_filters, advanced_meter_group_by_filters
						FROM features
						WHERE id = $1
					`, multipleFeatureId.String()).Scan(&originalFilters, &advancedFilters)
					require.NoError(t, err)

					require.True(t, originalFilters.Valid, "Original filters should not be NULL")
					require.JSONEq(t, `{"region":"us-east-1","environment":"production","team":"backend"}`, originalFilters.String, "Original filters should remain unchanged")
					require.True(t, advancedFilters.Valid, "Advanced filters should not be NULL")
					require.JSONEq(t, `{"region":{"$eq":"us-east-1"},"environment":{"$eq":"production"},"team":{"$eq":"backend"}}`, advancedFilters.String, "Advanced filters should have $eq wrappers")

					// Verify that the advanced_meter_group_by_filters column exists
					var columnExists bool
					err = db.QueryRow(`
						SELECT EXISTS (
							SELECT 1 
							FROM information_schema.columns 
							WHERE table_name = 'features' 
							AND column_name = 'advanced_meter_group_by_filters'
						)
					`).Scan(&columnExists)
					require.NoError(t, err)
					require.True(t, columnExists, "advanced_meter_group_by_filters column should exist after migration")
				},
			},
		},
	}.Test(t)
}
