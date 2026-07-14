package migrate_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	ommigrate "github.com/openmeterio/openmeter/tools/migrate"
	"github.com/openmeterio/openmeter/tools/migrate/legacyent"
)

func TestLegacyEntAdoptionSchemaParity(t *testing.T) {
	// given:
	// - one database migrated from scratch through the Atlas migration history
	// - one database migrated through the frozen Ent schema, reconciliation, and the remaining Atlas history
	// when:
	// - Atlas compares their resulting schemas
	// then:
	// - the adoption path has the same relational schema as a from-scratch migration
	// - programmable database objects and migration-owned seed state also match
	canonicalDB := testutils.InitPostgresDB(t)
	defer canonicalDB.Close(t)
	adoptedDB := testutils.InitPostgresDB(t)
	defer adoptedDB.Close(t)

	canonicalMigrator, err := ommigrate.New(ommigrate.MigrateOptions{
		ConnectionString: canonicalDB.URL,
		Migrations:       ommigrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	defer canonicalMigrator.CloseOrLogError()
	require.NoError(t, canonicalMigrator.Up())

	require.NoError(t, legacyent.MigrateToBaseline(t.Context(), adoptedDB.PGDriver.DB()))
	require.NoError(t, legacyent.Reconcile(t.Context(), adoptedDB.PGDriver.DB()))

	adoptedMigrator, err := ommigrate.New(ommigrate.MigrateOptions{
		ConnectionString: adoptedDB.URL,
		Migrations:       ommigrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	defer adoptedMigrator.CloseOrLogError()
	require.NoError(t, adoptedMigrator.Force(legacyent.BaselineVersion))
	require.NoError(t, adoptedMigrator.Up())

	canonicalVersion, canonicalDirty, err := canonicalMigrator.Version()
	require.NoError(t, err)
	adoptedVersion, adoptedDirty, err := adoptedMigrator.Version()
	require.NoError(t, err)
	require.False(t, canonicalDirty)
	require.False(t, adoptedDirty)
	require.Equal(t, canonicalVersion, adoptedVersion)

	canonicalAtlas, err := postgres.Open(canonicalDB.PGDriver.DB())
	require.NoError(t, err)
	adoptedAtlas, err := postgres.Open(adoptedDB.PGDriver.DB())
	require.NoError(t, err)

	canonicalSchemaName := currentSchema(t, canonicalDB.PGDriver.DB())
	adoptedSchemaName := currentSchema(t, adoptedDB.PGDriver.DB())
	require.Equal(t, canonicalSchemaName, adoptedSchemaName)

	inspectOptions := &schema.InspectOptions{Exclude: []string{"distributed_locks"}}
	canonicalSchema, err := canonicalAtlas.InspectSchema(t.Context(), canonicalSchemaName, inspectOptions)
	require.NoError(t, err)
	adoptedSchema, err := adoptedAtlas.InspectSchema(t.Context(), adoptedSchemaName, inspectOptions)
	require.NoError(t, err)

	changes, err := adoptedAtlas.SchemaDiff(adoptedSchema, canonicalSchema)
	require.NoError(t, err)
	if len(changes) != 0 {
		plan, err := adoptedAtlas.PlanChanges(t.Context(), "legacy_ent_adoption_parity", changes)
		require.NoError(t, err)
		require.Emptyf(t, changes, "adopted schema differs from canonical Atlas schema:\n%s", formatMigrationPlan(plan))
	}

	require.Equal(
		t,
		loadProgrammableDatabaseObjects(t, canonicalDB.PGDriver.DB(), canonicalSchemaName),
		loadProgrammableDatabaseObjects(t, adoptedDB.PGDriver.DB(), adoptedSchemaName),
	)

	var canonicalInvoiceWriteSchemaLevel int
	require.NoError(t, canonicalDB.PGDriver.DB().QueryRowContext(
		t.Context(),
		`SELECT schema_level FROM billing_invoice_write_schema_levels WHERE id = 'write_schema_level'`,
	).Scan(&canonicalInvoiceWriteSchemaLevel))
	var adoptedInvoiceWriteSchemaLevel int
	require.NoError(t, adoptedDB.PGDriver.DB().QueryRowContext(
		t.Context(),
		`SELECT schema_level FROM billing_invoice_write_schema_levels WHERE id = 'write_schema_level'`,
	).Scan(&adoptedInvoiceWriteSchemaLevel))
	require.Equal(t, canonicalInvoiceWriteSchemaLevel, adoptedInvoiceWriteSchemaLevel)
}

type programmableDatabaseObject struct {
	Kind       string
	Name       string
	Definition string
}

func currentSchema(t *testing.T, db *sql.DB) string {
	t.Helper()

	var schemaName string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT current_schema()`).Scan(&schemaName))

	return schemaName
}

// loadProgrammableDatabaseObjects returns normalized definitions for the PostgreSQL objects that
// are required by OpenMeter but are not fully represented by Atlas's relational schema diff.
func loadProgrammableDatabaseObjects(t *testing.T, db *sql.DB, schemaName string) []programmableDatabaseObject {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT object_kind, object_name, object_definition
		FROM (
			SELECT
				'extension' AS object_kind,
				e.extname AS object_name,
				e.extname AS object_definition
			FROM pg_extension e

			UNION ALL

			SELECT
				CASE c.relkind WHEN 'm' THEN 'materialized_view' ELSE 'view' END AS object_kind,
				c.relname AS object_name,
				pg_get_viewdef(c.oid, true) AS object_definition
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1
			  AND c.relkind IN ('v', 'm')

			UNION ALL

			SELECT
				'function' AS object_kind,
				p.proname || '(' || pg_get_function_identity_arguments(p.oid) || ')' AS object_name,
				pg_get_functiondef(p.oid) AS object_definition
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE n.nspname = $1
			  AND NOT EXISTS (
				SELECT 1
				FROM pg_depend d
				WHERE d.classid = 'pg_proc'::regclass
				  AND d.objid = p.oid
				  AND d.deptype = 'e'
			  )

			UNION ALL

			SELECT
				'trigger' AS object_kind,
				c.relname || '.' || t.tgname AS object_name,
				pg_get_triggerdef(t.oid, true) AS object_definition
			FROM pg_trigger t
			JOIN pg_class c ON c.oid = t.tgrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1
			  AND NOT t.tgisinternal
		) objects
		ORDER BY object_kind, object_name
	`, schemaName)
	require.NoError(t, err)
	defer rows.Close()

	objects := make([]programmableDatabaseObject, 0)
	for rows.Next() {
		var object programmableDatabaseObject
		require.NoError(t, rows.Scan(&object.Kind, &object.Name, &object.Definition))
		objects = append(objects, object)
	}
	require.NoError(t, rows.Err())

	return objects
}

func formatMigrationPlan(plan *migrate.Plan) string {
	commands := make([]string, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		commands = append(commands, fmt.Sprintf("%s;", change.Cmd))
	}

	return strings.Join(commands, "\n")
}
