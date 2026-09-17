package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	subscriptionTaxConfigBackfillSeedVersion = 20260917100416
	subscriptionTaxConfigBackfillVersion     = 20260917124624
	subscriptionTaxConfigValidateVersion     = 20260917124625
)

func TestSubscriptionTaxConfigBackfillMigration(t *testing.T) {
	namespace := "subscription_tax_config_backfill_test"

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	systemSaaS := ulid.Make().String()
	autoSaaS := ulid.Make().String()
	customGeneral := ulid.Make().String()
	deadIaaS := ulid.Make().String()
	malformedMappings := ulid.Make().String()
	phaseID := ulid.Make().String()

	rTieBreak := ulid.Make().String()
	rAttachCustom := ulid.Make().String()
	rCreate := ulid.Make().String()
	rCreateShared := ulid.Make().String()
	rCreateOverDead := ulid.Make().String()
	rBehaviorOnly := ulid.Make().String()
	rNoTaxConfig := ulid.Make().String()
	rClean := ulid.Make().String()
	rJSONIDRestore := ulid.Make().String()
	rColumnsOnly := ulid.Make().String()

	runner{
		stops: stops{
			{
				version:   subscriptionTaxConfigBackfillSeedVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedSubscriptionTaxCode(t, db, namespace, systemSaaS, "saas_business", "SaaS Business",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, `{"managed_by":"system"}`, t2, sql.NullTime{})

					seedSubscriptionTaxCode(t, db, namespace, autoSaaS, "stripe_txcd_10103001", "Stripe txcd_10103001",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, nil, t1, sql.NullTime{})

					seedSubscriptionTaxCode(t, db, namespace, customGeneral, "my-general-services", "General Electronically Supplied",
						`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil, t1, sql.NullTime{})

					seedSubscriptionTaxCode(t, db, namespace, deadIaaS, "stripe_txcd_20060051", "Stripe txcd_20060051",
						`[{"app_type":"stripe","tax_code":"txcd_20060051"}]`, nil, t1,
						sql.NullTime{Time: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true})
					seedSubscriptionTaxCode(t, db, namespace, malformedMappings, "malformed_mappings", "Malformed mappings",
						`{"unexpected":true}`, nil, t1, sql.NullTime{})

					seedSubscription(t, db, namespace, phaseID)
					seedSubscriptionItem(t, db, namespace, phaseID, rTieBreak, "tie_break", nil, nil,
						`{"behavior":"exclusive","stripe":{"code":"txcd_10103001"}}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rAttachCustom, "attach_custom", nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rCreate, "create", nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rCreateShared, "create_shared", nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rCreateOverDead, "create_over_dead", nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_20060051"}}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rBehaviorOnly, "behavior_only", nil, nil,
						`{"behavior":"inclusive"}`)
					seedSubscriptionItem(t, db, namespace, phaseID, rNoTaxConfig, "no_tax_config", nil, nil, nil)
					seedSubscriptionItem(t, db, namespace, phaseID, rClean, "clean", customGeneral, "exclusive",
						fmt.Sprintf(`{"behavior":"exclusive","stripe":{"code":"txcd_10000000"},"tax_code_id":%q}`, customGeneral))
					seedSubscriptionItem(t, db, namespace, phaseID, rJSONIDRestore, "json_id_restore", nil, nil,
						fmt.Sprintf(`{"behavior":"exclusive","tax_code_id":%q}`, customGeneral))
					seedSubscriptionItem(t, db, namespace, phaseID, rColumnsOnly, "columns_only", customGeneral, "inclusive", nil)
				},
			},
			{
				version:   subscriptionTaxConfigBackfillVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertSubscriptionTaxConstraintsValidated(t, db, false)

					assertSubscriptionItemTax(t, db, rTieBreak, systemSaaS)
					assertSubscriptionItemBehavior(t, db, rTieBreak, "exclusive")

					assertSubscriptionItemTax(t, db, rAttachCustom, customGeneral)
					assertSubscriptionItemBehavior(t, db, rAttachCustom, "inclusive")

					createdCode := subscriptionTaxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
					assertSubscriptionItemTax(t, db, rCreate, createdCode)
					assertSubscriptionItemTax(t, db, rCreateShared, createdCode)
					assertSubscriptionTaxCodeCreatedShape(t, db, createdCode, "txcd_40010001")

					createdIaaS := subscriptionTaxCodeIDByKey(t, db, namespace, "stripe_txcd_20060051")
					require.NotEqual(t, deadIaaS, createdIaaS)
					assertSubscriptionItemTax(t, db, rCreateOverDead, createdIaaS)
					assertSubscriptionTaxCodeCreatedShape(t, db, createdIaaS, "txcd_20060051")

					var deadAt sql.NullTime
					err := db.QueryRowContext(t.Context(), `SELECT deleted_at FROM tax_codes WHERE id = $1`, deadIaaS).Scan(&deadAt)
					require.NoError(t, err)
					require.True(t, deadAt.Valid)

					var behaviorOnlyCode sql.NullString
					err = db.QueryRowContext(t.Context(), `SELECT tax_code_id FROM subscription_items WHERE id = $1`, rBehaviorOnly).Scan(&behaviorOnlyCode)
					require.NoError(t, err)
					require.False(t, behaviorOnlyCode.Valid)
					assertSubscriptionItemBehavior(t, db, rBehaviorOnly, "inclusive")

					var taxConfigIsNull bool
					err = db.QueryRowContext(t.Context(), `SELECT tax_config IS NULL FROM subscription_items WHERE id = $1`, rNoTaxConfig).Scan(&taxConfigIsNull)
					require.NoError(t, err)
					require.True(t, taxConfigIsNull)

					assertSubscriptionItemTax(t, db, rClean, customGeneral)
					assertSubscriptionItemBehavior(t, db, rClean, "exclusive")
					assertSubscriptionItemTax(t, db, rJSONIDRestore, customGeneral)
					assertSubscriptionItemBehavior(t, db, rJSONIDRestore, "exclusive")
					assertSubscriptionItemTax(t, db, rColumnsOnly, customGeneral)
					assertSubscriptionItemBehavior(t, db, rColumnsOnly, "inclusive")

					var created int
					err = db.QueryRowContext(t.Context(), `
						SELECT count(*)
						FROM tax_codes
						WHERE namespace = $1
						  AND deleted_at IS NULL
						  AND key IN ('stripe_txcd_40010001', 'stripe_txcd_20060051')
					`, namespace).Scan(&created)
					require.NoError(t, err)
					require.Equal(t, 2, created)

					_, err = insertSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "invalid_stripe_write", nil, nil,
						`{"stripe":{"code":"txcd_30060006"}}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "subscription_item_tax_code_consistency")

					_, err = insertSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "invalid_behavior_write", nil, nil,
						`{"behavior":"exclusive"}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "subscription_item_tax_behavior_consistency")

					_, err = insertSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "missing_json_tax_code", customGeneral, nil, nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "subscription_item_tax_code_consistency")

					_, err = insertSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "missing_json_behavior", nil, "exclusive", nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "subscription_item_tax_behavior_consistency")

					seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "optional_after_migration", nil, nil, nil)
				},
			},
			{
				version:   subscriptionTaxConfigValidateVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertSubscriptionTaxConstraintsValidated(t, db, true)
				},
			},
			{
				version:   subscriptionTaxConfigBackfillSeedVersion,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					assertSubscriptionTaxConstraintsAbsent(t, db)
					assertSubscriptionItemTax(t, db, rTieBreak, systemSaaS)
					assertSubscriptionItemBehavior(t, db, rTieBreak, "exclusive")

					var created int
					err := db.QueryRowContext(t.Context(), `
						SELECT count(*)
						FROM tax_codes
						WHERE namespace = $1
						  AND deleted_at IS NULL
						  AND key IN ('stripe_txcd_40010001', 'stripe_txcd_20060051')
					`, namespace).Scan(&created)
					require.NoError(t, err)
					require.Equal(t, 2, created)
				},
			},
		},
	}.Test(t)
}

func TestSubscriptionTaxConfigBackfillMigrationFailsOnNonStripeCode(t *testing.T) {
	db, migrator := newSubscriptionTaxConfigBackfillTestEnv(t)
	namespace := "subscription_tax_config_backfill_invalid"
	phaseID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(subscriptionTaxConfigBackfillSeedVersion))
	seedSubscription(t, db, namespace, phaseID)
	seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "invalid", nil, nil,
		`{"stripe":{"code":"definitely-not-a-stripe-code"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-Stripe tax code")
}

func TestSubscriptionTaxConfigBackfillMigrationFailsOnOrphanedAutoKey(t *testing.T) {
	db, migrator := newSubscriptionTaxConfigBackfillTestEnv(t)
	namespace := "subscription_tax_config_backfill_orphaned"
	phaseID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(subscriptionTaxConfigBackfillSeedVersion))
	seedSubscriptionTaxCode(t, db, namespace, ulid.Make().String(), "stripe_txcd_40010001", "Repurposed",
		`[{"app_type":"stripe","tax_code":"txcd_99999999"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedSubscription(t, db, namespace, phaseID)
	seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "orphaned", nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "orphaned")
}

func TestSubscriptionTaxConfigBackfillMigrationFailsOnConflictingRepresentations(t *testing.T) {
	db, migrator := newSubscriptionTaxConfigBackfillTestEnv(t)
	namespace := "subscription_tax_config_backfill_conflict"
	phaseID := ulid.Make().String()
	taxCodeID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(subscriptionTaxConfigBackfillSeedVersion))
	seedSubscriptionTaxCode(t, db, namespace, taxCodeID, "general", "General",
		`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedSubscription(t, db, namespace, phaseID)
	seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "conflict", taxCodeID, "exclusive",
		fmt.Sprintf(`{"behavior":"inclusive","tax_code_id":%q}`, ulid.Make().String()))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflicting tax_code_id representations")
	require.Contains(t, err.Error(), "conflicting tax behavior representations")
}

func TestSubscriptionTaxConfigBackfillMigrationFailsOnMismatchedTaxIdentity(t *testing.T) {
	db, migrator := newSubscriptionTaxConfigBackfillTestEnv(t)
	namespace := "subscription_tax_config_backfill_mismatched_identity"
	phaseID := ulid.Make().String()
	taxCodeID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(subscriptionTaxConfigBackfillSeedVersion))
	seedSubscriptionTaxCode(t, db, namespace, taxCodeID, "general", "General",
		`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedSubscription(t, db, namespace, phaseID)
	seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "mismatched_identity", nil, nil,
		fmt.Sprintf(`{"stripe":{"code":"txcd_20060051"},"tax_code_id":%q}`, taxCodeID))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match the referenced live tax code Stripe app mapping")
}

func TestSubscriptionTaxConfigBackfillMigrationFailsOnInvalidNormalizedReference(t *testing.T) {
	testCases := []struct {
		name             string
		taxCodeNamespace string
		deletedAt        sql.NullTime
	}{
		{
			name:             "cross namespace",
			taxCodeNamespace: "another_namespace",
		},
		{
			name:             "deleted",
			taxCodeNamespace: "subscription_tax_config_backfill_invalid_reference",
			deletedAt: sql.NullTime{
				Time:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
				Valid: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, migrator := newSubscriptionTaxConfigBackfillTestEnv(t)
			namespace := "subscription_tax_config_backfill_invalid_reference"
			phaseID := ulid.Make().String()
			taxCodeID := ulid.Make().String()

			require.NoError(t, migrator.Migrate(subscriptionTaxConfigBackfillSeedVersion))
			seedSubscriptionTaxCode(t, db, tc.taxCodeNamespace, taxCodeID, "general", "General", nil, nil,
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), tc.deletedAt)
			seedSubscription(t, db, namespace, phaseID)
			seedSubscriptionItem(t, db, namespace, phaseID, ulid.Make().String(), "invalid_reference", taxCodeID, nil,
				fmt.Sprintf(`{"tax_code_id":%q}`, taxCodeID))

			err := migrator.Up()
			require.Error(t, err)
			require.Contains(t, err.Error(), "reference a missing, deleted, or cross-namespace tax code")
		})
	}
}

func newSubscriptionTaxConfigBackfillTestEnv(t *testing.T) (*sql.DB, *migrate.Migrate) {
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

func seedSubscriptionTaxCode(t *testing.T, db *sql.DB, namespace, id, key, name string, appMappings, annotations any, createdAt time.Time, deletedAt sql.NullTime) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO tax_codes (
			id, namespace, created_at, updated_at, deleted_at,
			name, key, app_mappings, annotations
		) VALUES (
			$1, $2, $3, $3, $4,
			$5, $6, $7::jsonb, $8::jsonb
		)`,
		id, namespace, createdAt, deletedAt,
		name, key, appMappings, annotations)
	require.NoError(t, err)
}

func seedSubscription(t *testing.T, db *sql.DB, namespace, phaseID string) {
	t.Helper()

	customerID := ulid.Make().String()
	subscriptionID := ulid.Make().String()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO customers (
			id, namespace, created_at, updated_at, key, name, currency
		) VALUES (
			$1, $2, NOW(), NOW(), 'tax-migration-customer', 'Tax migration customer', 'USD'
		)`, customerID, namespace)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO subscriptions (
			id, namespace, created_at, updated_at, active_from,
			customer_id, currency, billing_anchor, billing_cadence, pro_rating_config
		) VALUES (
			$1, $2, NOW(), NOW(), '2024-01-01',
			$3, 'USD', '2024-01-01', 'P1M', '{"enabled":true,"mode":"prorate_prices"}'::jsonb
		)`, subscriptionID, namespace, customerID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO subscription_phases (
			id, namespace, created_at, updated_at,
			name, key, subscription_id, active_from
		) VALUES (
			$1, $2, NOW(), NOW(),
			'Tax migration phase', 'default', $3, '2024-01-01'
		)`, phaseID, namespace, subscriptionID)
	require.NoError(t, err)
}

func seedSubscriptionItem(t *testing.T, db *sql.DB, namespace, phaseID, id, key string, taxCodeID, taxBehavior, taxConfig any) {
	t.Helper()

	_, err := insertSubscriptionItem(t, db, namespace, phaseID, id, key, taxCodeID, taxBehavior, taxConfig)
	require.NoError(t, err)
}

func insertSubscriptionItem(t *testing.T, db *sql.DB, namespace, phaseID, id, key string, taxCodeID, taxBehavior, taxConfig any) (sql.Result, error) {
	t.Helper()

	return db.ExecContext(t.Context(), `
		INSERT INTO subscription_items (
			id, namespace, created_at, updated_at, active_from,
			name, key, phase_id,
			tax_code_id, tax_behavior, tax_config
		) VALUES (
			$1, $2, NOW(), NOW(), '2024-01-01',
			$3, $3, $4,
			$5, $6, $7::jsonb
		)`, id, namespace, key, phaseID, taxCodeID, taxBehavior, taxConfig)
}

func subscriptionTaxCodeIDByKey(t *testing.T, db *sql.DB, namespace, key string) string {
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

func assertSubscriptionItemTax(t *testing.T, db *sql.DB, itemID, wantTaxCodeID string) {
	t.Helper()

	var taxCodeID, embeddedTaxCodeID string
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_code_id, tax_config ->> 'tax_code_id'
		FROM subscription_items
		WHERE id = $1
	`, itemID).Scan(&taxCodeID, &embeddedTaxCodeID)
	require.NoError(t, err)
	require.Equal(t, wantTaxCodeID, taxCodeID)
	require.Equal(t, wantTaxCodeID, embeddedTaxCodeID)
}

func assertSubscriptionItemBehavior(t *testing.T, db *sql.DB, itemID, wantBehavior string) {
	t.Helper()

	var behavior, embeddedBehavior sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_behavior, tax_config ->> 'behavior'
		FROM subscription_items
		WHERE id = $1
	`, itemID).Scan(&behavior, &embeddedBehavior)
	require.NoError(t, err)

	if wantBehavior == "" {
		require.False(t, behavior.Valid)
		require.False(t, embeddedBehavior.Valid)
		return
	}

	require.True(t, behavior.Valid)
	require.True(t, embeddedBehavior.Valid)
	require.Equal(t, wantBehavior, behavior.String)
	require.Equal(t, wantBehavior, embeddedBehavior.String)
}

func assertSubscriptionTaxConstraintsValidated(t *testing.T, db *sql.DB, wantValidated bool) {
	t.Helper()

	var total, validated int
	err := db.QueryRowContext(t.Context(), `
		SELECT
			count(*),
			count(*) FILTER (WHERE convalidated)
		FROM pg_constraint
		WHERE conrelid = 'subscription_items'::regclass
		  AND conname IN (
			'subscription_item_tax_code_consistency',
			'subscription_item_tax_behavior_consistency'
		  )
	`).Scan(&total, &validated)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	if wantValidated {
		require.Equal(t, 2, validated)
	} else {
		require.Zero(t, validated)
	}
}

func assertSubscriptionTaxConstraintsAbsent(t *testing.T, db *sql.DB) {
	t.Helper()

	var count int
	err := db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM pg_constraint
		WHERE conrelid = 'subscription_items'::regclass
		  AND conname IN (
			'subscription_item_tax_code_consistency',
			'subscription_item_tax_behavior_consistency'
		  )
	`).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}

func assertSubscriptionTaxCodeCreatedShape(t *testing.T, db *sql.DB, id, stripeCode string) {
	t.Helper()

	var name string
	var annotations sql.NullString
	var mappingType, mappingCode string
	err := db.QueryRowContext(t.Context(), `
		SELECT
			name,
			annotations,
			app_mappings -> 0 ->> 'app_type',
			app_mappings -> 0 ->> 'tax_code'
		FROM tax_codes
		WHERE id = $1
	`, id).Scan(&name, &annotations, &mappingType, &mappingCode)
	require.NoError(t, err)
	require.Equal(t, stripeCode, name)
	require.False(t, annotations.Valid)
	require.Equal(t, "stripe", mappingType)
	require.Equal(t, stripeCode, mappingCode)
}
