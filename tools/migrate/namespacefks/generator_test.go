package namespacefks_test

import (
	"database/sql"
	"strings"
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/stretchr/testify/require"

	entmigrate "github.com/openmeterio/openmeter/openmeter/ent/db/migrate"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate/namespacefks"
)

func TestGenerateBillingNamespaceForeignKeys(t *testing.T) {
	// given:
	// - the generated Ent schema and the billing tables that persist configuration and invoice snapshots
	// when:
	// - namespace guard SQL is generated from their existing ID-only foreign keys
	// then:
	// - all 13 eligible billing relationships and the missing parent index are represented
	output, err := namespacefks.Generate(namespacefks.GenerateInput{
		Tables: entmigrate.Tables,
		ChildTables: []string{
			"billing_profiles",
			"billing_customer_overrides",
			"billing_invoices",
		},
	})
	require.NoError(t, err)

	sql := string(output)
	require.Equal(t, 13, strings.Count(sql, "ADD CONSTRAINT"))
	require.Equal(t, 1, strings.Count(sql, "CREATE UNIQUE INDEX"))

	for _, relationship := range []string{
		`FOREIGN KEY ("namespace", "billing_profile_id")`,
		`FOREIGN KEY ("namespace", "customer_id")`,
		`FOREIGN KEY ("namespace", "invoicing_app_id")`,
		`FOREIGN KEY ("namespace", "payment_app_id")`,
		`FOREIGN KEY ("namespace", "source_billing_profile_id")`,
		`FOREIGN KEY ("namespace", "tax_app_id")`,
		`FOREIGN KEY ("namespace", "tax_code_id")`,
		`FOREIGN KEY ("namespace", "workflow_config_id")`,
	} {
		require.Contains(t, sql, relationship)
	}

	require.Contains(t, sql, `CREATE UNIQUE INDEX "billing_workflow_configs_namespace_id_key"`)
}

func TestGenerateRejectsUnknownChildTable(t *testing.T) {
	// given:
	// - an unknown child table selection
	// when:
	// - namespace guard SQL generation starts
	// then:
	// - generation fails instead of silently producing an incomplete overlay
	_, err := namespacefks.Generate(namespacefks.GenerateInput{
		Tables:      entmigrate.Tables,
		ChildTables: []string{"missing_table"},
	})
	require.EqualError(t, err, `child table "missing_table" not found`)
}

func TestGeneratedNamespaceForeignKeyGuardsCrossNamespaceReferences(t *testing.T) {
	// given:
	// - two namespace-owned tables connected by an ID-only foreign key with ON DELETE SET NULL
	// when:
	// - the generated namespace guard is applied
	// then:
	// - cross-namespace writes fail and the original delete behavior remains intact
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.Close(t)

	_, err := testDB.PGDriver.DB().Exec(`
		CREATE TABLE parents (
			id text PRIMARY KEY,
			namespace text NOT NULL
		);
		CREATE TABLE children (
			id text PRIMARY KEY,
			namespace text NOT NULL,
			parent_id text,
			CONSTRAINT children_parent_fk
				FOREIGN KEY (parent_id) REFERENCES parents (id) ON DELETE SET NULL
		);
	`)
	require.NoError(t, err)

	parentID := &schema.Column{Name: "id"}
	parentNamespace := &schema.Column{Name: "namespace"}
	parent := &schema.Table{
		Name:       "parents",
		Columns:    []*schema.Column{parentID, parentNamespace},
		PrimaryKey: []*schema.Column{parentID},
	}
	childID := &schema.Column{Name: "id"}
	childNamespace := &schema.Column{Name: "namespace"}
	childParentID := &schema.Column{Name: "parent_id", Nullable: true}
	child := &schema.Table{
		Name:       "children",
		Columns:    []*schema.Column{childID, childNamespace, childParentID},
		PrimaryKey: []*schema.Column{childID},
		ForeignKeys: []*schema.ForeignKey{{
			Symbol:     "children_parent_fk",
			Columns:    []*schema.Column{childParentID},
			RefTable:   parent,
			RefColumns: []*schema.Column{parentID},
		}},
	}

	generated, err := namespacefks.Generate(namespacefks.GenerateInput{
		Tables:      []*schema.Table{parent, child},
		ChildTables: []string{"children"},
	})
	require.NoError(t, err)

	_, err = testDB.PGDriver.DB().Exec(string(generated))
	require.NoError(t, err)

	_, err = testDB.PGDriver.DB().Exec(`
		INSERT INTO parents (id, namespace)
		VALUES ('parent-a', 'namespace-a'), ('parent-b', 'namespace-b')
	`)
	require.NoError(t, err)

	_, err = testDB.PGDriver.DB().Exec(`
		INSERT INTO children (id, namespace, parent_id)
		VALUES ('invalid-child', 'namespace-a', 'parent-b')
	`)
	require.Error(t, err)

	_, err = testDB.PGDriver.DB().Exec(`
		INSERT INTO children (id, namespace, parent_id)
		VALUES ('valid-child', 'namespace-a', 'parent-a')
	`)
	require.NoError(t, err)

	_, err = testDB.PGDriver.DB().Exec(`DELETE FROM parents WHERE id = 'parent-a'`)
	require.NoError(t, err)

	var (
		namespace string
		parentRef sql.NullString
	)
	err = testDB.PGDriver.DB().QueryRow(`
		SELECT namespace, parent_id
		FROM children
		WHERE id = 'valid-child'
	`).Scan(&namespace, &parentRef)
	require.NoError(t, err)
	require.Equal(t, "namespace-a", namespace)
	require.False(t, parentRef.Valid)
}
