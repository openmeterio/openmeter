// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var ErrNoChange = migrate.ErrNoChange

const (
	MigrationsTable = "schema_om"
)

// goMigrate is a type alias to avoid conflicts with embedding the Migrate struct vs calling Migrate.Migrate()
type goMigrate = migrate.Migrate

type Migrate struct {
	*goMigrate

	sourceDriver source.Driver
	logger       *slog.Logger
}

func (m *Migrate) LatestVersion() (uint, error) {
	version, err := m.sourceDriver.First()
	if err != nil {
		// Should not happen, as it means we don't have any migrations
		return 0, err
	}

	for {
		nextVersion, err := m.sourceDriver.Next(version)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return version, nil
			}
		}

		version = nextVersion
	}
}

func (m *Migrate) Up() error {
	return m.filterErrNoChange(m.goMigrate.Up())
}

func (m *Migrate) Down() error {
	return m.filterErrNoChange(m.goMigrate.Down())
}

func (m *Migrate) Migrate(version uint) error {
	return m.filterErrNoChange(m.goMigrate.Migrate(version))
}

func (m *Migrate) filterErrNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		m.logger.Info("no migrations to apply")
		return nil
	}

	return err
}

type logger struct {
	log *slog.Logger
}

var _ migrate.Logger = &logger{}

func (l *logger) Printf(format string, v ...interface{}) {
	l.log.Info(format, v...)
}

func (l *logger) Verbose() bool {
	return true
}

func NewLogger(log *slog.Logger) migrate.Logger {
	return &logger{log: log}
}

//go:embed migrations
var omMigrations embed.FS

type MigrationsConfig struct {
	FS             fs.FS
	FSPath         string
	StateTableName string
}

func (m *MigrationsConfig) Validate() error {
	var errs []error
	if m.FS == nil {
		errs = append(errs, errors.New("fs is required"))
	}
	if m.FSPath == "" {
		errs = append(errs, errors.New("fs path is required"))
	}

	if m.StateTableName == "" {
		errs = append(errs, errors.New("state table name is required"))
	}

	return errors.Join(errs...)
}

var OMMigrationsConfig = MigrationsConfig{
	FS:             omMigrations,
	FSPath:         "migrations",
	StateTableName: MigrationsTable,
}

type MigrateOptions struct {
	ConnectionString string
	Migrations       MigrationsConfig
	Logger           *slog.Logger
}

func (m *MigrateOptions) Validate() error {
	var errs []error

	if m.ConnectionString == "" {
		errs = append(errs, errors.New("connection string is required"))
	}

	if err := m.Migrations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("migrations config is invalid: %w", err))
	}

	if m.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

// New creates a new migrate instance.
func New(options MigrateOptions) (*Migrate, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	fs := NewSourceWrapper(options.Migrations.FS)
	sourceDriver, err := iofs.New(fs, options.Migrations.FSPath)
	if err != nil {
		return nil, err
	}

	conn, err := setMigrationTableName(options.ConnectionString, options.Migrations.StateTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to set migration table name: %w", err)
	}

	migrate, err := migrate.NewWithSourceInstance("iofs", sourceDriver, conn)
	if err != nil {
		return nil, err
	}

	return &Migrate{
		goMigrate:    migrate,
		sourceDriver: sourceDriver,
		logger:       options.Logger,
	}, nil
}

type WaitForMigrationOption func(*waitForMigrationOptions)

type waitForMigrationOptions struct {
	MaxRetries uint
	Delay      time.Duration
}

func getWaitForMigrationOptions(waitOpts []WaitForMigrationOption) waitForMigrationOptions {
	opts := waitForMigrationOptions{
		MaxRetries: 30,
		Delay:      1 * time.Second,
	}
	for _, opt := range waitOpts {
		opt(&opts)
	}

	return opts
}

func (m *Migrate) WaitForMigrationJob(waitOpts ...WaitForMigrationOption) error {
	opts := getWaitForMigrationOptions(waitOpts)

	latestKnownVersion, err := m.LatestVersion()
	if err != nil {
		return err
	}

	maxMigrationWaitDuration := time.Duration(opts.MaxRetries) * opts.Delay

	failAfter := time.Now().Add(maxMigrationWaitDuration)
	for {
		if time.Now().After(failAfter) {
			return fmt.Errorf("timeout waiting for migration job after %s", maxMigrationWaitDuration)
		}

		ver, dirty, err := m.Version()
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("database is dirty, please run migrations manually")
		}

		if ver >= latestKnownVersion {
			// We are at least on the version that is required by the binary => good to go
			m.logger.Info("database migrations are ready", "current_db_version", ver, "latest_known_version", latestKnownVersion)
			return nil
		}

		m.logger.Info("waiting for migration job",
			slog.Int("current_db_version", int(ver)),
			slog.Int("latest_known_version", int(latestKnownVersion)),
		)

		time.Sleep(opts.Delay)
	}
}

func setMigrationTableName(conn, tableName string) (string, error) {
	parsedURL, err := url.Parse(conn)
	if err != nil {
		return "", err
	}

	values := parsedURL.Query()
	values.Set("x-migrations-table", tableName)
	parsedURL.RawQuery = values.Encode()

	return parsedURL.String(), nil
}

func (m *Migrate) CloseOrLogError() {
	sourceErr, dbErr := m.goMigrate.Close()

	if sourceErr != nil {
		m.logger.Error("failed to close migration source", "error", sourceErr)
	}

	if dbErr != nil {
		m.logger.Error("failed to close postgres database", "error", dbErr)
	}
}
