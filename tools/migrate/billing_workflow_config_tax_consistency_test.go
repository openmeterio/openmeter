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
	billingWorkflowTaxConsistencySeedVersion     = 20260917124625
	billingWorkflowTaxConsistencyVersion         = 20260917150828
	billingWorkflowTaxConsistencyValidateVersion = 20260917150829
)

func TestBillingWorkflowTaxConsistencyMigration(t *testing.T) {
	namespace := "billing_workflow_tax_consistency_test"

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	systemSaaS := ulid.Make().String()
	autoSaaS := ulid.Make().String()
	customGeneral := ulid.Make().String()
	deadIaaS := ulid.Make().String()
	malformedMappings := ulid.Make().String()

	wTieBreak := ulid.Make().String()
	wAttachCustom := ulid.Make().String()
	wCreate := ulid.Make().String()
	wCreateShared := ulid.Make().String()
	wCreateOverDead := ulid.Make().String()
	wBehaviorOnly := ulid.Make().String()
	wNoTaxConfig := ulid.Make().String()
	wClean := ulid.Make().String()
	wJSONIDRestore := ulid.Make().String()
	wColumnsOnly := ulid.Make().String()

	runner{
		stops: stops{
			{
				version:   billingWorkflowTaxConsistencySeedVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedTaxCode(t, db, namespace, systemSaaS, "saas_business", "SaaS Business",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, `{"managed_by":"system"}`, t2, sql.NullTime{})
					seedTaxCode(t, db, namespace, autoSaaS, "stripe_txcd_10103001", "Stripe txcd_10103001",
						`[{"app_type":"stripe","tax_code":"txcd_10103001"}]`, nil, t1, sql.NullTime{})
					seedTaxCode(t, db, namespace, customGeneral, "my-general-services", "General Electronically Supplied",
						`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil, t1, sql.NullTime{})
					seedTaxCode(t, db, namespace, deadIaaS, "stripe_txcd_20060051", "Stripe txcd_20060051",
						`[{"app_type":"stripe","tax_code":"txcd_20060051"}]`, nil, t1,
						sql.NullTime{Time: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true})
					seedTaxCode(t, db, namespace, malformedMappings, "malformed_mappings", "Malformed mappings",
						`{"unexpected":true}`, nil, t1, sql.NullTime{})

					seedBillingWorkflowConfig(t, db, namespace, wTieBreak, nil, nil,
						`{"behavior":"exclusive","stripe":{"code":"txcd_10103001"}}`)
					seedBillingWorkflowConfig(t, db, namespace, wAttachCustom, nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_10000000"}}`)
					seedBillingWorkflowConfig(t, db, namespace, wCreate, nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedBillingWorkflowConfig(t, db, namespace, wCreateShared, nil, nil,
						`{"stripe":{"code":"txcd_40010001"}}`)
					seedBillingWorkflowConfig(t, db, namespace, wCreateOverDead, nil, nil,
						`{"behavior":"inclusive","stripe":{"code":"txcd_20060051"}}`)
					seedBillingWorkflowConfig(t, db, namespace, wBehaviorOnly, nil, nil,
						`{"behavior":"inclusive"}`)
					seedBillingWorkflowConfig(t, db, namespace, wNoTaxConfig, nil, nil, nil)
					seedBillingWorkflowConfig(t, db, namespace, wClean, customGeneral, "exclusive",
						fmt.Sprintf(`{"behavior":"exclusive","stripe":{"code":"txcd_10000000"},"tax_code_id":%q}`, customGeneral))
					seedBillingWorkflowConfig(t, db, namespace, wJSONIDRestore, nil, nil,
						fmt.Sprintf(`{"behavior":"exclusive","tax_code_id":%q}`, customGeneral))
					seedBillingWorkflowConfig(t, db, namespace, wColumnsOnly, customGeneral, "inclusive", nil)
				},
			},
			{
				version:   billingWorkflowTaxConsistencyVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertBillingWorkflowTaxConstraintsValidated(t, db, false)

					assertWorkflowConfigTax(t, db, wTieBreak, systemSaaS)
					assertBillingWorkflowBehavior(t, db, wTieBreak, "exclusive")
					assertWorkflowConfigTax(t, db, wAttachCustom, customGeneral)
					assertBillingWorkflowBehavior(t, db, wAttachCustom, "inclusive")

					createdCode := taxCodeIDByKey(t, db, namespace, "stripe_txcd_40010001")
					assertWorkflowConfigTax(t, db, wCreate, createdCode)
					assertWorkflowConfigTax(t, db, wCreateShared, createdCode)
					assertTaxCodeCreatedShape(t, db, createdCode, "txcd_40010001")

					createdIaaS := taxCodeIDByKey(t, db, namespace, "stripe_txcd_20060051")
					require.NotEqual(t, deadIaaS, createdIaaS)
					assertWorkflowConfigTax(t, db, wCreateOverDead, createdIaaS)
					assertTaxCodeCreatedShape(t, db, createdIaaS, "txcd_20060051")

					var behaviorOnlyCode sql.NullString
					err := db.QueryRowContext(t.Context(), `
						SELECT tax_code_id
						FROM billing_workflow_configs
						WHERE id = $1
					`, wBehaviorOnly).Scan(&behaviorOnlyCode)
					require.NoError(t, err)
					require.False(t, behaviorOnlyCode.Valid)
					assertBillingWorkflowBehavior(t, db, wBehaviorOnly, "inclusive")

					var taxConfigIsNull bool
					err = db.QueryRowContext(t.Context(), `
						SELECT invoice_default_tax_settings IS NULL
						FROM billing_workflow_configs
						WHERE id = $1
					`, wNoTaxConfig).Scan(&taxConfigIsNull)
					require.NoError(t, err)
					require.True(t, taxConfigIsNull)

					assertWorkflowConfigTax(t, db, wClean, customGeneral)
					assertBillingWorkflowBehavior(t, db, wClean, "exclusive")
					assertWorkflowConfigTax(t, db, wJSONIDRestore, customGeneral)
					assertBillingWorkflowBehavior(t, db, wJSONIDRestore, "exclusive")
					assertWorkflowConfigTax(t, db, wColumnsOnly, customGeneral)
					assertBillingWorkflowBehavior(t, db, wColumnsOnly, "inclusive")

					_, err = insertBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
						`{"stripe":{"code":"txcd_30060006"}}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "billing_workflow_config_tax_code_consistency")

					_, err = insertBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
						`{"behavior":"exclusive"}`)
					require.Error(t, err)
					require.Contains(t, err.Error(), "billing_workflow_config_tax_behavior_consistency")

					_, err = insertBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), customGeneral, nil, nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "billing_workflow_config_tax_code_consistency")

					_, err = insertBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, "exclusive", nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "billing_workflow_config_tax_behavior_consistency")

					seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil, nil)
					seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, "exclusive",
						`{"behavior":"exclusive"}`)
				},
			},
			{
				version:   billingWorkflowTaxConsistencyValidateVersion,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertBillingWorkflowTaxConstraintsValidated(t, db, true)
				},
			},
			{
				version:   billingWorkflowTaxConsistencySeedVersion,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					assertBillingWorkflowTaxConstraintsAbsent(t, db)
					assertWorkflowConfigTax(t, db, wTieBreak, systemSaaS)
					assertBillingWorkflowBehavior(t, db, wTieBreak, "exclusive")

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

func TestBillingWorkflowTaxConsistencyMigrationFailsOnNonStripeCode(t *testing.T) {
	db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
	namespace := "billing_workflow_tax_consistency_invalid"

	require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
	seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		`{"stripe":{"code":"definitely-not-a-stripe-code"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-Stripe tax code")
}

func TestBillingWorkflowTaxConsistencyMigrationFailsOnOrphanedAutoKey(t *testing.T) {
	db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
	namespace := "billing_workflow_tax_consistency_orphaned"

	require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
	seedTaxCode(t, db, namespace, ulid.Make().String(), "stripe_txcd_40010001", "Repurposed",
		`[{"app_type":"stripe","tax_code":"txcd_99999999"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		`{"stripe":{"code":"txcd_40010001"}}`)

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "orphaned")
}

func TestBillingWorkflowTaxConsistencyMigrationFailsOnConflictingRepresentations(t *testing.T) {
	db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
	namespace := "billing_workflow_tax_consistency_conflict"
	taxCodeID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
	seedTaxCode(t, db, namespace, taxCodeID, "general", "General",
		`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), taxCodeID, "exclusive",
		fmt.Sprintf(`{"behavior":"inclusive","tax_code_id":%q}`, ulid.Make().String()))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "conflicting tax_code_id representations")
	require.Contains(t, err.Error(), "conflicting tax behavior representations")
}

func TestBillingWorkflowTaxConsistencyMigrationFailsOnMismatchedTaxIdentity(t *testing.T) {
	db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
	namespace := "billing_workflow_tax_consistency_mismatched_identity"
	taxCodeID := ulid.Make().String()

	require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
	seedTaxCode(t, db, namespace, taxCodeID, "general", "General",
		`[{"app_type":"stripe","tax_code":"txcd_10000000"}]`, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), sql.NullTime{})
	seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		fmt.Sprintf(`{"stripe":{"code":"txcd_20060051"},"tax_code_id":%q}`, taxCodeID))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match the referenced live tax code Stripe app mapping")
}

func TestBillingWorkflowTaxConsistencyMigrationFailsOnUnresolvedEmbeddedID(t *testing.T) {
	db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
	namespace := "billing_workflow_tax_consistency_unresolved"

	require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
	seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), nil, nil,
		fmt.Sprintf(`{"tax_code_id":%q}`, ulid.Make().String()))

	err := migrator.Up()
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be resolved to a live tax code in the same namespace")
}

func TestBillingWorkflowTaxConsistencyMigrationFailsOnInvalidNormalizedReference(t *testing.T) {
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
			taxCodeNamespace: "billing_workflow_tax_consistency_invalid_reference",
			deletedAt: sql.NullTime{
				Time:  time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
				Valid: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, migrator := newBillingWorkflowTaxConsistencyTestEnv(t)
			namespace := "billing_workflow_tax_consistency_invalid_reference"
			taxCodeID := ulid.Make().String()

			require.NoError(t, migrator.Migrate(billingWorkflowTaxConsistencySeedVersion))
			seedTaxCode(t, db, tc.taxCodeNamespace, taxCodeID, "general", "General", nil, nil,
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), tc.deletedAt)
			seedBillingWorkflowConfig(t, db, namespace, ulid.Make().String(), taxCodeID, nil,
				fmt.Sprintf(`{"tax_code_id":%q}`, taxCodeID))

			err := migrator.Up()
			require.Error(t, err)
			require.Contains(t, err.Error(), "reference a missing, deleted, or cross-namespace tax code")
		})
	}
}

func newBillingWorkflowTaxConsistencyTestEnv(t *testing.T) (*sql.DB, *migrate.Migrate) {
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

func seedBillingWorkflowConfig(t *testing.T, db *sql.DB, namespace, id string, taxCodeID, taxBehavior, taxSettings any) {
	t.Helper()

	_, err := insertBillingWorkflowConfig(t, db, namespace, id, taxCodeID, taxBehavior, taxSettings)
	require.NoError(t, err)
}

func insertBillingWorkflowConfig(t *testing.T, db *sql.DB, namespace, id string, taxCodeID, taxBehavior, taxSettings any) (sql.Result, error) {
	t.Helper()

	return db.ExecContext(t.Context(), `
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
}

func assertBillingWorkflowBehavior(t *testing.T, db *sql.DB, workflowConfigID, wantBehavior string) {
	t.Helper()

	var behavior, embeddedBehavior sql.NullString
	err := db.QueryRowContext(t.Context(), `
		SELECT tax_behavior, invoice_default_tax_settings ->> 'behavior'
		FROM billing_workflow_configs
		WHERE id = $1
	`, workflowConfigID).Scan(&behavior, &embeddedBehavior)
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

func assertBillingWorkflowTaxConstraintsValidated(t *testing.T, db *sql.DB, wantValidated bool) {
	t.Helper()

	var total, validated int
	err := db.QueryRowContext(t.Context(), `
		SELECT
			count(*),
			count(*) FILTER (WHERE convalidated)
		FROM pg_constraint
		WHERE conrelid = 'billing_workflow_configs'::regclass
		  AND conname IN (
			'billing_workflow_config_tax_code_consistency',
			'billing_workflow_config_tax_behavior_consistency'
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

func assertBillingWorkflowTaxConstraintsAbsent(t *testing.T, db *sql.DB) {
	t.Helper()

	var count int
	err := db.QueryRowContext(t.Context(), `
		SELECT count(*)
		FROM pg_constraint
		WHERE conrelid = 'billing_workflow_configs'::regclass
		  AND conname IN (
			'billing_workflow_config_tax_code_consistency',
			'billing_workflow_config_tax_behavior_consistency'
		  )
	`).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}
