package testutils

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"

	entschema "github.com/openmeterio/openmeter/openmeter/ent"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type PostgresDBState uint8

const (
	PostgresDBStateEmpty PostgresDBState = iota
	PostgresDBStateEntMigrated
	PostgresDBStateAtlasMigrated
)

type emptyMigrator struct{}

func (emptyMigrator) Hash() (string, error) {
	return common.NewRecursiveHash(
		common.Field("migrator", "openmeter-empty-v1"),
	).String(), nil
}

func (emptyMigrator) Migrate(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}

type entMigrator struct{}

func (entMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash(
		common.Field("migrator", "openmeter-ent-v1"),
	)
	hash.Add([]byte(entschema.GeneratedMigrationSchema()))

	return hash.String(), nil
}

func (entMigrator) Migrate(ctx context.Context, db *sql.DB, _ pgtestdb.Config) error {
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := entdb.NewClient(entdb.Driver(driver))

	return client.Schema.Create(ctx)
}

type atlasMigrator struct {
	Migrations migrate.MigrationsConfig
	Logger     *slog.Logger
}

func newAtlasMigrator(t testing.TB) atlasMigrator {
	t.Helper()

	return atlasMigrator{
		Migrations: migrate.OMMigrationsConfig,
		Logger:     NewLogger(t),
	}
}

func (m atlasMigrator) Hash() (string, error) {
	migrations := m.Migrations
	if migrations.FS == nil {
		migrations = migrate.OMMigrationsConfig
	}

	if err := migrations.Validate(); err != nil {
		return "", err
	}

	hash := common.NewRecursiveHash(
		common.Field("migrator", "openmeter-atlas-v1"),
		common.Field("fsPath", migrations.FSPath),
		common.Field("stateTableName", migrations.StateTableName),
	)

	atlasSum, err := fs.ReadFile(migrations.FS, path.Join(migrations.FSPath, "atlas.sum"))
	if err != nil {
		return "", err
	}
	hash.Add(atlasSum)

	return hash.String(), nil
}

func (m atlasMigrator) Migrate(
	_ context.Context,
	_ *sql.DB,
	templateConf pgtestdb.Config,
) (err error) {
	migrations := m.Migrations
	if migrations.FS == nil {
		migrations = migrate.OMMigrationsConfig
	}

	if m.Logger == nil {
		return errors.New("logger is required")
	}

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: templateConf.URL(),
		Migrations:       migrations,
		Logger:           m.Logger,
	})
	if err != nil {
		return err
	}

	defer func() {
		srcErr, dbErr := migrator.Close()
		err = errors.Join(err, srcErr, dbErr)
	}()

	return migrator.Up()
}

type TestDB struct {
	EntDriver *entdriver.EntPostgresDriver
	PGDriver  *pgdriver.Driver
	URL       string
}

func (d *TestDB) Close(t testing.TB) {
	if err := d.EntDriver.Close(); err != nil {
		t.Errorf("failed to close Ent driver: %v", err)
	}

	if err := d.PGDriver.Close(); err != nil {
		t.Errorf("failed to close Postgres driver: %v", err)
	}
}

func InitPostgresDB(t testing.TB, state PostgresDBState) *TestDB {
	t.Helper()

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	config := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "postgres",
		Host:       os.Getenv("POSTGRES_HOST"),
		Port:       port,
		Options:    "sslmode=disable",
	}

	if config.Host == "" {
		t.Skip("postgres host is not set. Set POSTGRES_HOST to run database tests")
	}

	var migrator pgtestdb.Migrator
	switch state {
	case PostgresDBStateEmpty:
		migrator = emptyMigrator{}
	case PostgresDBStateEntMigrated:
		migrator = entMigrator{}
	case PostgresDBStateAtlasMigrated:
		migrator = newAtlasMigrator(t)
	default:
		t.Fatalf("unsupported Postgres database state: %d", state)
		return nil
	}

	dbConf := pgtestdb.Custom(t, config, migrator)
	if dbConf == nil {
		t.Fatalf("failed to get db config")
		return nil
	}

	postgresDriver, err := pgdriver.NewPostgresDriver(
		t.Context(),
		dbConf.URL(),
	)
	if err != nil {
		t.Fatalf("failed to get postgres driver: %s", err)
	}

	entDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())

	return &TestDB{
		PGDriver:  postgresDriver,
		EntDriver: entDriver,
		URL:       dbConf.URL(),
	}
}
