package postgres_connector

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	"github.com/peterldowns/pgtestdb"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/migrate"
)

// Open new connection
func Open(databaseURL string) (*db.Client, error) {
	// TODO: inject trace and metrics provider
	database, err := otelsql.Open("pgx", databaseURL, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, err
	}

	drv := entsql.OpenDB(dialect.Postgres, database)

	return db.NewClient(db.Driver(drv)), nil
}

// EntMigrator is a migrator for pgtestdb.
type EntMigrator struct{}

// Hash returns the md5 hash of the schema file.
func (m *EntMigrator) Hash() (string, error) {
	return "", nil
}

// Migrate shells out to the `atlas` CLI program to migrate the template
// database.
//
//	atlas schema apply --auto-approve --url $DB --to file://$schemaFilePath
func (m *EntMigrator) Migrate(
	ctx context.Context,
	db *sql.DB,
	templateConf pgtestdb.Config,
) error {
	driver := entsql.OpenDB(dialect.Postgres, db)
	schema := migrate.NewSchema(driver)
	return schema.Create(ctx)
}

// Prepare is a no-op method.
func (*EntMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (*EntMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}
