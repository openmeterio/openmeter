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
	URL       string
}

func (d *TestDB) Close(t *testing.T) {
	if err := d.EntDriver.Close(); err != nil {
		t.Errorf("failed to close Ent driver: %v", err)
	}

	if err := d.PGDriver.Close(); err != nil {
		t.Errorf("failed to close Postgres driver: %v", err)
	}
}

type options struct {
	config        pgtestdb.Config
	migrator      pgtestdb.Migrator
	driverOptions []pgdriver.Option
}

type Option interface {
	apply(*options)
}

type optionFunc func(c *options)

func (fn optionFunc) apply(c *options) {
	fn(c)
}

func WithPostgresConfig(config pgtestdb.Config) Option {
	return optionFunc(func(o *options) {
		o.config = config
	})
}

func WithMigrator(migrator pgtestdb.Migrator) Option {
	return optionFunc(func(o *options) {
		o.migrator = migrator
	})
}

func WithDriverOptions(opts []pgdriver.Option) Option {
	return optionFunc(func(o *options) {
		o.driverOptions = opts
	})
}

func InitPostgresDB(t *testing.T, opts ...Option) *TestDB {
	t.Helper()

	o := options{
		config: pgtestdb.Config{
			DriverName: "pgx",
			User:       "postgres",
			Password:   "postgres",
			Host:       os.Getenv("POSTGRES_HOST"),
			Port:       "5432",
			Options:    "sslmode=disable",
		},
		migrator: &NoopMigrator{}, // TODO: fix migrations
	}

	for _, opt := range opts {
		opt.apply(&o)
	}

	if o.config.Host == "" {
		t.Skip("postgres host is not set. Either set POSTGRES_HOST environment variable or use WithPostgresConfig option")
	}

	dbConf := pgtestdb.Custom(t, o.config, o.migrator)
	if dbConf == nil {
		t.Fatalf("failed to get db config")
	}

	postgresDriver, err := pgdriver.NewPostgresDriver(
		t.Context(),
		dbConf.URL(),
		o.driverOptions...,
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
