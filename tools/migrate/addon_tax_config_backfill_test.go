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
	addonTaxConfigBackfillSeedVersion = 20260916155504
	addonTaxConfigBackfillVersion     = 20260917064455
)

func TestAddonTaxConfigBackfillMigration(t *testing.T) {
	namespace := "addon_tax_config_backfill_test"

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	systemSaaS := ulid.Make().String()
	autoSaaS := ulid.Make().String()
	customGeneral := ulid.Make().String()
	deadIaaS := ulid.Make().String()
	addonID := ulid.Make().String()

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
				version:   addonTaxConfigBackfillSeedVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedAddonTaxCode(t, db, namespace, systemSaaS, "saas_business", "SaaS Business",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, `{"managed_by":"system"}`, t2, sql.NullTime{})

					seedAddonTaxCode(t, db, namespace, autoSaaS, "stripe_txcd_10103001", "Stripe txcd_10103001",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, nil, t1, sql.NullTime{})

					seedAddonTaxCode(t, db, namespace, customGeneral, "my-general-services", "General Electronically Supplied",
						`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil, t1, sql.NullTime{})

					seedAddonTaxCode(t, db, namespace, deadIaaS, "stripe_txcd_20060051", "Stripe txcd_20060051",
						`[{"app_type":"stripe","tax_code":"txcd_20060051"}]`, nil, t1,
						sql.NullTime{Time: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true})

					seedAddon(t, db, namespace, addonID)
					seedAddonRateCard(t, db, namespace, addonID, rTieBreak, "tie_break", nil, nil,
						`{"behavior":"exclusive","stripe":{"code":"txcd_10103001"}}`)
					seedAddonRateCard(t, db, namespace, addonID, rAttachCustom, "attach_custom", nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
					seedAddonRateCard(t, db, namespace, addonID, rCreate, "create", nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedAddonRateCard(t, db, namespace, addonID, rCreateShared, "create_shared", nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedAddonRateCard(t, db, namespace, addonID, rCreateOverDead, "create_over_dead", nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_20060051"}}`)
					seedAddonRateCard(t, db, namespace, addonID, rBehaviorOnly, "behavior_only", nil, nil,
						`{"behavior":"inclusive"}`)
					seedAddonRateCard(t, db, namespace, addonID, rNoTaxConfig, "no_tax_config", nil, nil, nil)
					seedAddonRateCard(t, db, namespace, addonID, rClean, "clean", customGeneral, "exclusive",
						fmt.Sprintf(`{"behavior":"exclusive","stripe":{"code":"txcd_10000000"},"tax_code_id":%q}`, customGeneral))
					seedAddonRateCard(t, db, namespace, addonID, rJSONIDRestore, "json_id_restore", nil, nil,
						fmt.Sprintf(`{"behavior":"exclusive","tax_code_id":%q}`, customGeneral))
					seedAddonRateCard(t, db, namespace, addonID, rColumnsOnly, "columns_only", customGeneral, "inclusive", nil)
				},
			},
			{
				version:   addonTaxConfigBackfillVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertAddonRateCardTax(t, db, rTieBreak, systemSaaS)
					assertAddonRateCardBehavior(t, db, rTieBreak, "exclusive")

					assertAddonRateCardTax(t, db, rAttachCustom, customGeneral)
					assertAddonRateCardBehavior(t, db, rAttachCustom, "inclusive")

					createdCode := addonTaxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
					assertAddonRateCardTax(t, db, rCreate, createdCode)
					assertAddonRateCardTax(t, db, rCreateShared, createdCode)
					assertAddonTaxCodeCreatedShape(t, db, createdCode, "txcd_40010001")

					createdIaaS := addonTaxCodeIDByKey(t, db, namespace, "stripe_txcd_20060051")
					require.NotEqual(t, deadIaaS, createdIaaS)
					assertAddonRateCardTax(t, db, rCreateOverDead, createdIaaS)
					assertAddonTaxCodeCreatedShape(t, db, createdIaaS, "txcd_20060051")

					var deadAt sql.NullTime
					err := db.QueryRowContext(t.Context(), `SELECT deleted_at FROM tax_codes WHERE id = $1`, deadIaaS).Scan(&deadAt)
					require.NoError(t, err)
					require.True(t, deadAt.Valid)

					var behaviorOnlyCode sql.NullString
					err = db.QueryRowContext(t.Context(), `SELECT tax_code_id FROM addon_rate_cards WHERE id = $1`, rBehaviorOnly).Scan(&behaviorOnlyCode)
					require.NoError(t, err)
					require.False(t, behaviorOnlyCode.Valid)
					assertAddonRateCardBehavior(t, db, rBehaviorOnly, "inclusive")

					var taxConfigIsNull bool
					err = db.QueryRowContext(t.Context(), `SELECT tax_config IS NULL FROM addon_rate_cards WHERE id = $1`, rNoTaxConfig).Scan(&taxConfigIsNull)
					require.NoError(t, err)
					require.True(t, taxConfigIsNull)

					assertAddonRateCardTax(t, db, rClean, customGeneral)
					assertAddonRateCardBehavior(t, db, rClean, "exclusive")
					assertAddonRateCardTax(t, db, rJSONIDRestore, customGeneral)
					assertAddonRateCardBehavior(t, db, rJSONIDRestore, "exclusive")
					assertAddonRateCardTax(t, db, rColumnsOnly, customGeneral)
					assertAddonRateCardBehavior(t, db, rColumnsOnly, "inclusive")

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

					_, err = insertAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "invalid_stripe_write", nil, nil,
						`{"stripe":{"code":"txcd_30060006"}}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "addon_rate_card_tax_code_consistency")

					_, err = insertAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "invalid_behavior_write", nil, nil,
						`{"behavior":"exclusive"}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "addon_rate_card_tax_behavior_consistency")

					_, err = insertAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "missing_json_tax_code", customGeneral, nil, nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "addon_rate_card_tax_code_consistency")

					_, err = insertAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "missing_json_behavior", nil, "exclusive", nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "addon_rate_card_tax_behavior_consistency")

					seedAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "optional_after_migration", nil, nil, nil)
				},
			},
		},
	}.Test(t)
}

func TestAddonTaxConfigBackfillMigrationFailsOnNonStripeCode(t *testing.T) {
	db, migrator := newAddonTaxConfigBackfillTestEnv(t)
	namespace := "addon_tax_config_backfill_invalid"
	addonID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(addonTaxConfigBackfillSeedVersion))
	seedAddon(t, db, namespace, addonID)
	seedAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "invalid", nil, nil,
		`{"stripe":{"code":"definitely-not-a-stripe-code"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-Stripe tax code")
}

func TestAddonTaxConfigBackfillMigrationFailsOnOrphanedAutoKey(t *testing.T) {
	db, migrator := newAddonTaxConfigBackfillTestEnv(t)
	namespace := "addon_tax_config_backfill_orphaned"
	addonID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(addonTaxConfigBackfillSeedVersion))
	seedAddonTaxCode(t, db, namespace, ulid.Make().String(), "stripe_txcd_40010001", "Repurposed",
		`[{"app_type":"stripe","tax_code":"txcd_99999999"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedAddon(t, db, namespace, addonID)
	seedAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "orphaned", nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "orphaned")
}

func TestAddonTaxConfigBackfillMigrationFailsOnConflictingRepresentations(t *testing.T) {
	db, migrator := newAddonTaxConfigBackfillTestEnv(t)
	namespace := "addon_tax_config_backfill_conflict"
	addonID := ulid.Make().String()
	taxCodeID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(addonTaxConfigBackfillSeedVersion))
	seedAddonTaxCode(t, db, namespace, taxCodeID, "general", "General",
		`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedAddon(t, db, namespace, addonID)
	seedAddonRateCard(t, db, namespace, addonID, ulid.Make().String(), "conflict", taxCodeID, "exclusive",
		fmt.Sprintf(`{"behavior":"inclusive","tax_code_id":%q}`, ulid.Make().String()))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflicting tax_code_id representations")
	require.Contains(t, err.Error(), "conflicting tax behavior representations")
}

func newAddonTaxConfigBackfillTestEnv(t *testing.T) (*sql.DB, *migrate.Migrate) {
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

func seedAddonTaxCode(t *testing.T, db *sql.DB, namespace, id, key, name string, appMappings, annotations any, createdAt time.Time, deletedAt sql.NullTime) {
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

func seedAddon(t *testing.T, db *sql.DB, namespace, id string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO addons (
			id, namespace, created_at, updated_at,
			name, key, version, currency
		) VALUES (
			$1, $2, NOW(), NOW(),
			'Tax migration add-on', 'tax_migration_addon', 1, 'USD'
		)`, id, namespace)
	require.NoError(t, err)
}

func seedAddonRateCard(t *testing.T, db *sql.DB, namespace, addonID, id, key string, taxCodeID, taxBehavior, taxConfig any) {
	t.Helper()

	_, err := insertAddonRateCard(t, db, namespace, addonID, id, key, taxCodeID, taxBehavior, taxConfig)
	require.NoError(t, err)
}

func insertAddonRateCard(t *testing.T, db *sql.DB, namespace, addonID, id, key string, taxCodeID, taxBehavior, taxConfig any) (sql.Result, error) {
	t.Helper()

	return db.ExecContext(t.Context(), `
		INSERT INTO addon_rate_cards (
			id, namespace, created_at, updated_at,
			name, key, type, addon_id,
			tax_code_id, tax_behavior, tax_config
		) VALUES (
			$1, $2, NOW(), NOW(),
			$3, $3, 'FLAT_FEE', $4,
			$5, $6, $7::jsonb
		)`, id, namespace, key, addonID, taxCodeID, taxBehavior, taxConfig)
}

func addonTaxCodeIDByKey(t *testing.T, db *sql.DB, namespace, key string) string {
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

func assertAddonRateCardTax(t *testing.T, db *sql.DB, rateCardID, wantTaxCodeID string) {
	t.Helper()

	var taxCodeID, embeddedTaxCodeID string
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_code_id, tax_config ->> 'tax_code_id'
		FROM addon_rate_cards
		WHERE id = $1
	`, rateCardID).Scan(&taxCodeID, &embeddedTaxCodeID)
	require.NoError(t, err)
	require.Equal(t, wantTaxCodeID, taxCodeID)
	require.Equal(t, wantTaxCodeID, embeddedTaxCodeID)
}

func assertAddonRateCardBehavior(t *testing.T, db *sql.DB, rateCardID, wantBehavior string) {
	t.Helper()

	var behavior, embeddedBehavior sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_behavior, tax_config ->> 'behavior'
		FROM addon_rate_cards
		WHERE id = $1
	`, rateCardID).Scan(&behavior, &embeddedBehavior)
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

func assertAddonTaxCodeCreatedShape(t *testing.T, db *sql.DB, id, stripeCode string) {
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
