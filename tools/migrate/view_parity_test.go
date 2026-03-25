package migrate_test

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
	"github.com/openmeterio/openmeter/tools/migrate/viewgen"
)

var (
	createViewRE        = regexp.MustCompile(`(?is)CREATE\s+VIEW\s+"([^"]+)"\s+AS\s*(.+?);`)
	viewMigrationStmtRE = regexp.MustCompile(`(?im)\b(?:CREATE|DROP)\s+(?:MATERIALIZED\s+)?VIEW\b`)
	whitespaceRE        = regexp.MustCompile(`\s+`)
)

type viewColumn struct {
	Name     string
	Position int
	DataType string
	UDTName  string
}

func stripSQLLineComments(sqlText string) string {
	var b strings.Builder
	for _, line := range strings.Split(sqlText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return b.String()
}

func parseExpectedViews(schemaSQL string) (map[string]string, error) {
	cleaned := stripSQLLineComments(schemaSQL)
	matches := createViewRE.FindAllStringSubmatch(cleaned, -1)
	if len(matches) == 0 {
		return map[string]string{}, nil
	}

	out := make(map[string]string, len(matches))
	for _, m := range matches {
		if len(m) != 3 {
			return nil, fmt.Errorf("unexpected create view match groups: %d", len(m))
		}

		name, body := m[1], strings.TrimSpace(m[2])
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("duplicate CREATE VIEW for %q in generated views SQL", name)
		}

		out[name] = body
	}

	return out, nil
}

func normalizeViewSQL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	s = whitespaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// stripViewStatements removes CREATE/DROP VIEW statements from a SQL migration
// while keeping all other statements intact. It splits the SQL by semicolons and
// filters out any statement that contains a VIEW keyword.
func stripViewStatements(sqlText string) string {
	parts := strings.Split(sqlText, ";")

	var kept []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if viewMigrationStmtRE.MatchString(trimmed) {
			continue
		}
		kept = append(kept, part)
	}

	if len(kept) == 0 {
		return ""
	}

	return strings.Join(kept, ";") + ";\n"
}

func buildMigrationsWithoutViews(cfg migrate.MigrationsConfig) (migrate.MigrationsConfig, error) {
	filtered := fstest.MapFS{}

	err := fs.WalkDir(cfg.FS, cfg.FSPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(cfg.FS, path)
		if err != nil {
			return err
		}

		data := b
		if strings.HasSuffix(path, ".sql") && viewMigrationStmtRE.Match(b) {
			stripped := stripViewStatements(string(b))
			if strings.TrimSpace(stripped) == "" {
				// Migration has only VIEW statements, exclude it entirely
				return nil
			}
			data = []byte(stripped)
		}

		filtered[path] = &fstest.MapFile{
			Data: append([]byte(nil), data...),
			Mode: 0o644,
		}

		return nil
	})
	if err != nil {
		return migrate.MigrationsConfig{}, err
	}

	return migrate.MigrationsConfig{
		FS:             filtered,
		FSPath:         cfg.FSPath,
		StateTableName: cfg.StateTableName,
	}, nil
}

func newMigratorForTest(t *testing.T, connectionString string, cfg migrate.MigrationsConfig) *migrate.Migrate {
	t.Helper()

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: connectionString,
		Migrations:       cfg,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)

	return migrator
}

func loadPublicViewDefinition(t *testing.T, db *sql.DB, viewName string) string {
	t.Helper()

	var def string
	err := db.QueryRow(`
		SELECT pg_get_viewdef(c.oid, false)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'v'
		  AND n.nspname = 'public'
		  AND c.relname = $1
	`, viewName).Scan(&def)
	require.NoError(t, err, "view %q should exist", viewName)

	return def
}

func loadPublicViewColumns(t *testing.T, db *sql.DB, viewName string) []viewColumn {
	t.Helper()

	rows, err := db.Query(`
		SELECT column_name, ordinal_position, data_type, udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		ORDER BY ordinal_position
	`, viewName)
	require.NoError(t, err)
	defer rows.Close()

	var out []viewColumn
	for rows.Next() {
		var c viewColumn
		err := rows.Scan(&c.Name, &c.Position, &c.DataType, &c.UDTName)
		require.NoError(t, err)
		out = append(out, c)
	}
	require.NoError(t, rows.Err())

	return out
}

func applyGeneratedViews(t *testing.T, db *sql.DB) {
	t.Helper()

	sql, err := viewgen.GenerateSQL("../../openmeter/ent/schema")
	require.NoError(t, err)

	cleaned := strings.TrimSpace(stripSQLLineComments(string(sql)))
	require.NotEmpty(t, cleaned, "generated views SQL should not be empty")

	_, err = db.Exec(cleaned)
	require.NoError(t, err)
}

func TestViewDefinitionsMatchGeneratedSchemaSQL(t *testing.T) {
	sql, err := viewgen.GenerateSQL("../../openmeter/ent/schema")
	require.NoError(t, err)

	expectedBodies, err := parseExpectedViews(string(sql))
	require.NoError(t, err)
	if len(expectedBodies) == 0 {
		t.Skip("generated views SQL defines no CREATE VIEW statements; nothing to validate")
	}

	manualDB := testutils.InitPostgresDB(t)
	defer manualDB.PGDriver.Close()

	generatedDB := testutils.InitPostgresDB(t)
	defer generatedDB.PGDriver.Close()

	manualMigrator := newMigratorForTest(t, manualDB.URL, migrate.OMMigrationsConfig)
	defer func() {
		srcErr, dbErr := manualMigrator.Close()
		require.NoError(t, errors.Join(srcErr, dbErr))
	}()

	filteredCfg, err := buildMigrationsWithoutViews(migrate.OMMigrationsConfig)
	require.NoError(t, err)

	generatedMigrator := newMigratorForTest(t, generatedDB.URL, filteredCfg)
	defer func() {
		srcErr, dbErr := generatedMigrator.Close()
		require.NoError(t, errors.Join(srcErr, dbErr))
	}()

	require.NoError(t, manualMigrator.Up())
	require.NoError(t, generatedMigrator.Up())
	applyGeneratedViews(t, generatedDB.PGDriver.DB())

	viewNames := make([]string, 0, len(expectedBodies))
	for name := range expectedBodies {
		viewNames = append(viewNames, name)
	}
	slices.Sort(viewNames)

	for _, viewName := range viewNames {
		t.Run(viewName, func(t *testing.T) {
			manualColumns := loadPublicViewColumns(t, manualDB.PGDriver.DB(), viewName)
			generatedColumns := loadPublicViewColumns(t, generatedDB.PGDriver.DB(), viewName)
			require.Equal(t, generatedColumns, manualColumns,
				"view %q columns differ between migrated DB and generated-schema DB", viewName)

			manualDef := normalizeViewSQL(loadPublicViewDefinition(t, manualDB.PGDriver.DB(), viewName))
			generatedDef := normalizeViewSQL(loadPublicViewDefinition(t, generatedDB.PGDriver.DB(), viewName))
			require.Equal(t, generatedDef, manualDef,
				"view %q definition differs between migrated DB and generated-schema DB\nmanual:    %s\ngenerated: %s",
				viewName, manualDef, generatedDef)
		})
	}
}
