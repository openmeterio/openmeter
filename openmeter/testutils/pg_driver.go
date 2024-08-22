package testutils

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	"github.com/peterldowns/pgtestdb"

	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

// NoopMigrator is a migrator for pgtestdb.
type NoopMigrator struct{}

// Hash returns the md5 hash of the schema file.
func (m *NoopMigrator) Hash() (string, error) {
	return "", nil
}

// Migrate shells out to the `atlas` CLI program to migrate the template
// database.
//
//	atlas schema apply --auto-approve --url $DB --to file://$schemaFilePath
func (m *NoopMigrator) Migrate(
	ctx context.Context,
	db *sql.DB,
	templateConf pgtestdb.Config,
) error {
	return nil
}

// Prepare is a no-op method.
func (*NoopMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (*NoopMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}

type TestDB struct {
	EntDriver *entdriver.EntPostgresDriver
	PGDriver  *pgdriver.Driver
	SQLDriver *sql.DB
	URL       string
}

func InitPostgresDB(t *testing.T) *TestDB {
	t.Helper()

	// Dagger will set the POSTGRES_HOST environment variable for `make test`.
	// If you need to run credit tests without Dagger you can set the POSTGRES_HOST environment variable.
	// For example to use the Postgres in docker compose you can run `POSTGRES_HOST=localhost go test ./internal/credit/...`
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		t.Skip("POSTGRES_HOST not set")
	}

	// TODO: fix migrations
	dbConf := pgtestdb.Custom(t, pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "postgres",
		Host:       host,
		Port:       "5432",
		Options:    "sslmode=disable",
	}, &NoopMigrator{})

	postgresDriver, err := pgdriver.NewPostgresDriver(
		context.TODO(),
		dbConf.URL(),
	)
	if err != nil {
		t.Fatalf("failed to get pg driver: %s", err)
	}

	entDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())

	return &TestDB{
		PGDriver:  postgresDriver,
		EntDriver: entDriver,
		URL:       dbConf.URL(),
	}
}
