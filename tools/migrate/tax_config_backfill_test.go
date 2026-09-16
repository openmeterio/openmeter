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
	// The last migration before the tax config backfill — fixtures are seeded here.
	taxConfigBackfillSeedVersion = 20260914072004
	// The tax config backfill migration itself.
	taxConfigBackfillVersion = 20260916155504
)

func TestTaxConfigBackfillMigration(t *testing.T) {
	namespace := "tax_config_backfill_test"

	// T1 < T2 so the older row loses to the system-managed row despite age.
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	// txcd_10103001 has two live owners: the system-managed seeded template (newer)
	// and an auto-created row (older). The tie-break must pick the system-managed one.
	systemSaas := ulid.Make().String()
	autoSaas := ulid.Make().String()

	// txcd_10000000 has one owner under a user-chosen key — the lookup is
	// mapping-based, so it is found and attached.
	customGeneral := ulid.Make().String()

	// txcd_20060051 is only owned by a soft-deleted entity — the lookup must skip
	// it and create a fresh live entity instead.
	deadIaas := ulid.Make().String()

	// Workflow config rows, one per scenario:
	wTieBreak := ulid.Make().String()       // stripe txcd_10103001 + behavior, no code
	wAttachCustom := ulid.Make().String()   // stripe txcd_10000000 + behavior, no code
	wCreate := ulid.Make().String()         // stripe txcd_40010001, no behavior
	wCreateOverDead := ulid.Make().String() // stripe txcd_20060051 + behavior
	wBehaviorOnly := ulid.Make().String()   // behavior only, no stripe
	wClean := ulid.Make().String()          // already dual-written, must stay untouched
	wJsonIDRestore := ulid.Make().String()  // JSON tax_code_id of a live entity, no stripe

	runner{
		stops: stops{
			{
				// Stop 1: seed fixtures after the last migration before the backfill.
				version:   taxConfigBackfillSeedVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedTaxCode(t, db, namespace, systemSaas, "saas_business", "SaaS Business",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, `{"managed_by":"system"}`, t2, sql.NullTime{})

					seedTaxCode(t, db, namespace, autoSaas, "stripe_txcd_10103001", "Stripe txcd_10103001 (auto)",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, nil, t1, sql.NullTime{})

					seedTaxCode(t, db, namespace, customGeneral, "my-general-services", "General Electronically Supplied",
						`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil, t1, sql.NullTime{})

					seedTaxCode(t, db, namespace, deadIaas, "stripe_txcd_20060051", "Stripe txcd_20060051 (auto)",
						`[{"app_type":"stripe","tax_code":"txcd_20060051"}]`, nil, t1,
						sql.NullTime{Time: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true})

					seedWorkflowConfig(t, db, namespace, wTieBreak, nil, nil,
						`{"behavior":"exclusive","stripe":{"code":"txcd_10103001"}}`)
					seedWorkflowConfig(t, db, namespace, wAttachCustom, nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
					seedWorkflowConfig(t, db, namespace, wCreate, nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedWorkflowConfig(t, db, namespace, wCreateOverDead, nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_20060051"}}`)
					seedWorkflowConfig(t, db, namespace, wBehaviorOnly, nil, nil,
						`{"behavior":"inclusive"}`)
					seedWorkflowConfig(t, db, namespace, wClean, customGeneral, "exclusive",
						fmt.Sprintf(`{"behavior":"exclusive","stripe":{"code":"txcd_10000000"},"tax_code_id":%q}`, customGeneral))
					seedWorkflowConfig(t, db, namespace, wJsonIDRestore, nil, nil,
						fmt.Sprintf(`{"behavior":"exclusive","tax_code_id":%q}`, customGeneral))
				},
			},
			{
				// Stop 2: after the backfill migration has run.
				version:   taxConfigBackfillVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// The tie-break row attaches to the system-managed entity, in both
					// the column and the embedded JSON scalar.
					assertWorkflowConfigTax(t, db, wTieBreak, systemSaas)
					assertWorkflowConfigBehavior(t, db, wTieBreak, "exclusive")

					// The user-created entity is found by its app mapping.
					assertWorkflowConfigTax(t, db, wAttachCustom, customGeneral)
					assertWorkflowConfigBehavior(t, db, wAttachCustom, "inclusive")

					// The unowned code gets a user-style entity: auto key, code as name,
					// no annotations, single stripe mapping.
					createdRice := taxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
					assertWorkflowConfigTax(t, db, wCreate, createdRice)
					assertTaxCodeCreatedShape(t, db, createdRice, "txcd_40010001")
					assertWorkflowConfigBehavior(t, db, wCreate, "") // no behavior in JSON

					// The soft-deleted owner is skipped; a fresh live entity is created.
					createdIaas := taxCodeIDByKey(t, db, namespace, "stripe_txcd_20060051")
					assertWorkflowConfigTax(t, db, wCreateOverDead, createdIaas)
					assertTaxCodeCreatedShape(t, db, createdIaas, "txcd_20060051")
					assertWorkflowConfigBehavior(t, db, wCreateOverDead, "inclusive")

					// The dead entity stays dead.
					var deadAt sql.NullTime
					err := db.QueryRow(`SELECT deleted_at FROM tax_codes WHERE id = $1`, deadIaas).Scan(&deadAt)
					require.NoError(t, err)
					require.True(t, deadAt.Valid, "the soft-deleted entity must not be revived")

					// The behavior-only row keeps NULL tax_code_id — behavior-only is
					// valid state, the migration must not invent a code for it.
					var behaviorOnlyCode sql.NullString
					err = db.QueryRow(`SELECT tax_code_id FROM billing_workflow_configs WHERE id = $1`, wBehaviorOnly).Scan(&behaviorOnlyCode)
					require.NoError(t, err)
					require.False(t, behaviorOnlyCode.Valid)
					assertWorkflowConfigBehavior(t, db, wBehaviorOnly, "inclusive")

					// The already-clean row is untouched.
					assertWorkflowConfigTax(t, db, wClean, customGeneral)
					assertWorkflowConfigBehavior(t, db, wClean, "exclusive")

					// The JSON tax_code_id restore: the column is stamped from the
					// entity the JSON already references.
					assertWorkflowConfigTax(t, db, wJsonIDRestore, customGeneral)
					assertWorkflowConfigBehavior(t, db, wJsonIDRestore, "exclusive")

					// Exactly two entities were created, both auto-keyed for this
					// migration's codes.
					var created int
					err = db.QueryRow(`
						SELECT count(*) FROM tax_codes
						WHERE namespace = $1 AND deleted_at IS NULL AND key LIKE 'stripe_%' AND key NOT IN ('stripe_txcd_10103001')
					`, namespace).Scan(&created)
					require.NoError(t, err)
					require.Equal(t, 2, created)
				},
			},
		},
	}.Test(t)
}

func TestTaxConfigBackfillMigrationFailsOnNonStripeCode(t *testing.T) {
	db, migrator := newTaxConfigBackfillTestEnv(t)
	namespace := "tax_config_backfill_invalid"

	require.NoError(t, migrator.Migrate(taxConfigBackfillSeedVersion))

	// A workflow config whose legacy JSON carries a code that is not a Stripe tax
	// code at all.
	seedWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		`{"stripe":{"code":"definitely-not-a-stripe-code"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-Stripe tax code")
}

func TestTaxConfigBackfillMigrationFailsOnOrphanedAutoKey(t *testing.T) {
	db, migrator := newTaxConfigBackfillTestEnv(t)
	namespace := "tax_config_backfill_orphaned"

	require.NoError(t, migrator.Migrate(taxConfigBackfillSeedVersion))

	// A live entity holding the auto key for txcd_40010001 whose mapping points
	// elsewhere — the Go auto-create path returns an orphaned-key error for this
	// state, and so must the migration.
	seedTaxCode(t, db, namespace, ulid.Make().String(), "stripe_txcd_40010001", "Repurposed",
		`[{"app_type":"stripe","tax_code":"txcd_99999999"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})

	seedWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "orphaned")
}

func newTaxConfigBackfillTestEnv(t *testing.T) (*sql.DB, *migrate.Migrate) {
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

func seedTaxCode(t *testing.T, db *sql.DB, namespace, id, key, name string, appMappings, annotations any, createdAt time.Time, deletedAt sql.NullTime) {
	t.Helper()

	_, err := db.Exec(`
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

func seedWorkflowConfig(t *testing.T, db *sql.DB, namespace, id string, taxCodeID, taxBehavior any, taxSettings string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO billing_workflow_configs (
			id, namespace, created_at, updated_at,
			collection_alignment, line_collection_period,
			invoice_auto_advance, invoice_draft_period,
			invoice_due_after, invoice_collection_method,
			invoice_progressive_billing, tax_code_id, tax_behavior,
			invoice_default_tax_settings
		) VALUES (
			$1, $2, NOW(), NOW(),
			'subscription', 'P1M',
			true, 'P1D',
			'P30D', 'charge_automatically',
			false, $3, $4,
			$5::jsonb
		)`,
		id, namespace, taxCodeID, taxBehavior, taxSettings)
	require.NoError(t, err)
}

func taxCodeIDByKey(t *testing.T, db *sql.DB, namespace, key string) string {
	t.Helper()

	var id string
	err := db.QueryRow(`
		SELECT id FROM tax_codes
		WHERE namespace = $1 AND key = $2 AND deleted_at IS NULL
	`, namespace, key).Scan(&id)
	require.NoError(t, err)

	return id
}

func assertWorkflowConfigTax(t *testing.T, db *sql.DB, workflowConfigID, wantTaxCodeID string) {
	t.Helper()

	var taxCodeID string
	err := db.QueryRow(`
		SELECT tax_code_id FROM billing_workflow_configs WHERE id = $1
	`, workflowConfigID).Scan(&taxCodeID)
	require.NoError(t, err)
	require.Equal(t, wantTaxCodeID, taxCodeID)

	var embeddedTaxCodeID string
	err = db.QueryRow(`
		SELECT invoice_default_tax_settings ->> 'tax_code_id' FROM billing_workflow_configs WHERE id = $1
	`, workflowConfigID).Scan(&embeddedTaxCodeID)
	require.NoError(t, err)
	require.Equal(t, wantTaxCodeID, embeddedTaxCodeID)
}

func assertWorkflowConfigBehavior(t *testing.T, db *sql.DB, workflowConfigID, wantBehavior string) {
	t.Helper()

	var behavior sql.NullString
	err := db.QueryRow(`
		SELECT tax_behavior FROM billing_workflow_configs WHERE id = $1
	`, workflowConfigID).Scan(&behavior)
	require.NoError(t, err)

	if wantBehavior == "" {
		require.False(t, behavior.Valid, "tax_behavior must stay NULL when the JSON has no behavior")
		return
	}

	require.True(t, behavior.Valid)
	require.Equal(t, wantBehavior, behavior.String)
}

func assertTaxCodeCreatedShape(t *testing.T, db *sql.DB, id, stripeCode string) {
	t.Helper()

	var name string
	var annotations sql.NullString
	err := db.QueryRow(`
		SELECT name, annotations FROM tax_codes WHERE id = $1
	`, id).Scan(&name, &annotations)
	require.NoError(t, err)
	require.Equal(t, stripeCode, name, "auto-created tax codes use the stripe code as their name")
	require.False(t, annotations.Valid, "auto-created tax codes carry no annotations")

	var mappingType, mappingCode string
	err = db.QueryRow(`
		SELECT app_mappings -> 0 ->> 'app_type', app_mappings -> 0 ->> 'tax_code'
		FROM tax_codes WHERE id = $1
	`, id).Scan(&mappingType, &mappingCode)
	require.NoError(t, err)
	require.Equal(t, "stripe", mappingType)
	require.Equal(t, stripeCode, mappingCode)
}
