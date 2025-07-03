package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sourcegraph/conc/pool"

	"github.com/openmeterio/openmeter/.dagger/internal/dagger"
)

func (m *Openmeter) Migrate() *Migrate {
	return &Migrate{
		Source: m.Source,
	}
}

type Migrate struct {
	Source *dagger.Directory
}

// GenerateSqlcTestdata creates a complete SQLC testdata directory for a given migration version
func (m *Migrate) GenerateSqlcTestdata(
	ctx context.Context,
	// Migration version (timestamp format like 20240826120919)
	version int,
) (*dagger.Directory, error) {
	versionStr := strconv.Itoa(version)

	// Create postgres service
	postgres := dag.Postgres(dagger.PostgresOpts{
		Version: postgresVersion,
		Name:    fmt.Sprintf("sqlc-gen-%s", versionStr),
	})

	// Use golang-migrate CLI directly
	migrateContainer := dag.Container().
		From("migrate/migrate:latest").
		WithServiceBinding("postgres", postgres.Service()).
		WithDirectory("/migrations", m.Source.Directory("tools/migrate/migrations")).
		WithExec([]string{
			"migrate",
			"-path", "/migrations",
			"-database", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable&x-migrations-table=schema_om",
			"goto", versionStr,
		})

	// Wait for migration to complete
	_, err := migrateContainer.Sync(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	// Create schema dump using pg_dump (without sleep, let it fail fast if not ready)
	schemaContainer := dag.Container().
		From("postgres:"+postgresVersion).
		WithServiceBinding("postgres", postgres.Service()).
		WithEnvVariable("PGPASSWORD", "postgres").
		WithExec([]string{
			"pg_dump",
			"-h", "postgres",
			"-U", "postgres",
			"-d", "postgres",
			"-s",
			"--no-owner",
			"--no-acl",
		})

	schemaDump, err := schemaContainer.Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("schema dump failed: %w", err)
	}

	// Create sqlc.yaml config
	sqlcConfig := `version: "2"
sql:
  - engine: "postgresql"
    queries: "sqlc/queries.sql"
    schema: "sqlc/db-schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
`

	// Create empty queries.sql placeholder for SQLC
	emptyQueries := `-- Add your SQL queries here
-- Example:
-- name: GetExampleByID :one
-- SELECT * FROM example_table WHERE id = $1;

-- Placeholder query for SQLC validation
-- name: GetSchemaVersion :one
SELECT version FROM schema_om ORDER BY version DESC LIMIT 1;
`

	// Build the result directory with sqlc.yaml, schema, and placeholder queries in sqlc/ subdirectory
	result := dag.Directory().
		WithNewFile("sqlc.yaml", sqlcConfig).
		WithNewFile("sqlc/db-schema.sql", schemaDump).
		WithNewFile("sqlc/queries.sql", emptyQueries)

	// Run sqlc generate to create Go types
	sqlcContainer := dag.Container().
		From("sqlc/sqlc:latest").
		WithDirectory("/work", result).
		WithWorkdir("/work").
		WithExec([]string{"/workspace/sqlc", "generate"})

	// Get the generated files from the container
	generatedFiles := sqlcContainer.Directory("/work")

	return generatedFiles, nil
}

func (m *Migrate) Check(ctx context.Context) error {
	app := goModuleCross("").
		WithSource(m.Source).
		Container().
		WithEnvVariable("GOFLAGS", "-tags=musl")

	bin := dag.Container(dagger.ContainerOpts{
		Platform: "linux/amd64",
	}).From(AtlasContainerImage).File("atlas")

	atlas := app.
		WithFile("/bin/atlas", bin).
		WithDirectory("openmeter/ent", m.Source.Directory("openmeter/ent")).
		WithDirectory("tools/migrate/migrations", m.Source.Directory("tools/migrate/migrations")).
		WithFile("atlas.hcl", m.Source.File("atlas.hcl"))

	p := pool.New().WithErrors().WithContext(ctx)

	// Always validate schema is generated
	p.Go(func(ctx context.Context) error {
		result := app.
			WithExec([]string{"go", "generate", "-x", "./openmeter/ent/..."}).
			Directory("openmeter/ent")

		source := m.Source.Directory("openmeter/ent")

		err := diff(ctx, source, result)
		if err != nil {
			return fmt.Errorf("schema is not in sync with generated code")
		}
		return nil
	})

	// Always validate migrations are in sync with schema
	p.Go(func(ctx context.Context) error {
		postgres := dag.Postgres(dagger.PostgresOpts{
			Version: postgresVersion,
			Name:    "no-diff",
		})

		result := atlas.
			WithServiceBinding("postgres", postgres.Service()).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "diff", "test"}).
			Directory("tools/migrate/migrations")

		source := m.Source.Directory("tools/migrate/migrations")
		err := diff(ctx, source, result)
		if err != nil {
			return fmt.Errorf("migrations are not in sync with schema")
		}

		return nil
	})

	// Always lint last 10 migrations
	p.Go(syncFunc(
		atlas.
			WithServiceBinding("postgres", dag.Postgres(dagger.PostgresOpts{
				Version: postgresVersion,
				Name:    "last-10",
			}).Service()).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "lint", "--latest", "10"}),
	))

	// Validate checksum is intact
	p.Go(syncFunc(
		atlas.
			WithServiceBinding("postgres", dag.Postgres(dagger.PostgresOpts{
				Version: postgresVersion,
				Name:    "validate",
			}).Service()).
			WithExec([]string{"atlas", "migrate", "--env", "ci", "validate"}),
	))

	return p.Wait()
}
