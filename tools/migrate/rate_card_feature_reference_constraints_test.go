package migrate_test

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type rateCardFeatureReferenceConstraintTable struct {
	name                           string
	parentColumn                   string
	parentID                       string
	featureReferenceConstraintName string
}

type rateCardFeatureReferenceConstraintRow struct {
	table      rateCardFeatureReferenceConstraintTable
	namespace  string
	key        string
	featureID  any
	featureKey any
}

func insertRateCardFeatureReferenceConstraintRow(
	t testing.TB,
	db *sql.DB,
	row rateCardFeatureReferenceConstraintRow,
) error {
	t.Helper()

	_, err := db.ExecContext(t.Context(), fmt.Sprintf(`
		INSERT INTO %s (
			id, namespace, created_at, updated_at, name, key, type,
			%s, feature_id, feature_key
		) VALUES (
			$1, $2, NOW(), NOW(), $3, $3, 'FLAT_FEE', $4, $5, $6
		)
	`, row.table.name, row.table.parentColumn),
		ulid.Make().String(),
		row.namespace,
		row.key,
		row.table.parentID,
		row.featureID,
		row.featureKey,
	)

	return err
}

func TestRateCardFeatureReferenceConstraintsMigration(t *testing.T) {
	t.Parallel()

	const (
		namespace       = "rate_card_feature_reference_constraints"
		previousVersion = uint(20260814114215)
		targetVersion   = uint(20260818090123)
	)

	// given:
	// - the completed rate-card backfill and both valid and incomplete feature references
	// when:
	// - the feature-reference constraint migration is applied and rolled back
	// then:
	// - existing incomplete rows block rollout, both-null and complete rows remain valid,
	//   and new incomplete or empty references are rejected until rollback
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
	featureID := ulid.Make().String()
	featureKey := "constraint_feature"
	emptyIDFeatureKey := "empty_id_constraint_feature"
	planID := ulid.Make().String()
	phaseID := ulid.Make().String()
	addonID := ulid.Make().String()

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO features (id, namespace, created_at, updated_at, name, key)
		VALUES
			($1, $2, NOW(), NOW(), 'Constraint feature', $3),
			('', $2, NOW(), NOW(), 'Empty ID constraint feature', $4)
	`, featureID, namespace, featureKey, emptyIDFeatureKey)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plans (
			id, namespace, created_at, updated_at, name, key, version,
			currency, billing_cadence, pro_rating_config
		) VALUES (
			$1, $2, NOW(), NOW(), 'Constraint plan', 'constraint_plan', 1,
			'USD', 'P1M', '{"enabled":true,"mode":"prorate_prices"}'::jsonb
		)
	`, planID, namespace)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO plan_phases (
			id, namespace, created_at, updated_at, name, key, plan_id, index
		) VALUES (
			$1, $2, NOW(), NOW(), 'Constraint phase', 'constraint_phase', $3, 0
		)
	`, phaseID, namespace, planID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO addons (
			id, namespace, created_at, updated_at, name, key, version, currency
		) VALUES (
			$1, $2, NOW(), NOW(), 'Constraint add-on', 'constraint_addon', 1, 'USD'
		)
	`, addonID, namespace)
	require.NoError(t, err)

	tables := []rateCardFeatureReferenceConstraintTable{
		{
			name:                           "plan_rate_cards",
			parentColumn:                   "phase_id",
			parentID:                       phaseID,
			featureReferenceConstraintName: "plan_rate_card_feature_reference",
		},
		{
			name:                           "addon_rate_cards",
			parentColumn:                   "addon_id",
			parentID:                       addonID,
			featureReferenceConstraintName: "addon_rate_card_feature_reference",
		},
	}

	for index, table := range tables {
		require.NoError(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
			table:      table,
			namespace:  namespace,
			key:        fmt.Sprintf("legacy_incomplete_%d", index),
			featureID:  featureID,
			featureKey: nil,
		}))
	}

	err = migrator.Migrate(targetVersion)
	require.Error(t, err, "existing incomplete rate cards must block the constraint migration")
	require.NoError(t, migrator.Force(previousVersion), "clear the failed migration marker after DDL rollback")

	for _, table := range tables {
		_, err = db.ExecContext(t.Context(), fmt.Sprintf(`DELETE FROM %s`, table.name))
		require.NoError(t, err)
	}
	require.NoError(t, migrator.Migrate(targetVersion))

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			require.NoError(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "without_feature",
				featureID:  nil,
				featureKey: nil,
			}))
			require.NoError(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "complete_feature",
				featureID:  featureID,
				featureKey: featureKey,
			}))

			require.Error(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "id_only",
				featureID:  featureID,
				featureKey: nil,
			}))
			require.Error(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "key_only",
				featureID:  nil,
				featureKey: featureKey,
			}))
			require.Error(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "empty_key",
				featureID:  featureID,
				featureKey: "",
			}))

			err := insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
				table:      table,
				namespace:  namespace,
				key:        "empty_id",
				featureID:  "",
				featureKey: emptyIDFeatureKey,
			})
			require.ErrorContains(t, err, table.featureReferenceConstraintName)
		})
	}

	require.NoError(t, migrator.Migrate(previousVersion))
	for index, table := range tables {
		require.NoError(t, insertRateCardFeatureReferenceConstraintRow(t, db, rateCardFeatureReferenceConstraintRow{
			table:      table,
			namespace:  namespace,
			key:        fmt.Sprintf("incomplete_after_rollback_%d", index),
			featureID:  featureID,
			featureKey: nil,
		}))
	}
}
