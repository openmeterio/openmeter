package postgres_connector

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/peterldowns/pgtestdb"
	"github.com/stretchr/testify/assert"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/migrate"
	meter_internal "github.com/openmeterio/openmeter/internal/meter"
)

func TestConnector(t *testing.T) {
	meterRepository := meter_internal.NewInMemoryRepository(nil)

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit_model.Connector, db_client *db.Client)
	}{
		{
			name:        "ImplementsInterface",
			description: "PostgresConnector implements feature.Connector interface",
			test: func(t *testing.T, connector credit_model.Connector, db_client *db.Client) {
				assert.Implements(t, (*credit_model.Connector)(nil), connector)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := initDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository)
			tc.test(t, connector, databaseClient)
		})
	}
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

func initDB(t *testing.T) *entsql.Driver {
	t.Helper()

	// Dagger will set the POSTGRES_HOST environment variable for `make test`.
	// If you need to run credit tests without Dagger you can set the POSTGRES_HOST environment variable.
	// For example to use the Postgres in docker compose you can run `POSTGRES_HOST=localhost go test ./internal/credit/...`
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		t.Skip("POSTGRES_HOST not set")
	}

	return entsql.OpenDB(dialect.Postgres, pgtestdb.New(t, pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "postgres",
		Host:       host,
		Port:       "5432",
		Options:    "sslmode=disable",
	}, &EntMigrator{}))
}
