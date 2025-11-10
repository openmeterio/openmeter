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

	"github.com/avast/retry-go/v4"
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
var OMMigrations embed.FS

// NewMigrate creates a new migrate instance.
func NewMigrate(conn string, fs fs.FS, fsPath string) (*Migrate, error) {
	fs = NewSourceWrapper(fs)
	sourceDriver, err := iofs.New(fs, fsPath)
	if err != nil {
		return nil, err
	}

	migrate, err := migrate.NewWithSourceInstance("iofs", sourceDriver, conn)
	if err != nil {
		return nil, err
	}

	return &Migrate{
		goMigrate:    migrate,
		sourceDriver: sourceDriver,
	}, nil
}

func getMigrationForConn(conn string) (*Migrate, error) {
	conn, err := SetMigrationTableName(conn, MigrationsTable)
	if err != nil {
		return nil, err
	}

	m, err := NewMigrate(conn, OMMigrations, "migrations")
	if err != nil {
		return nil, err
	}

	return m, nil
}

func Up(conn string) error {
	m, err := getMigrationForConn(conn)
	if err != nil {
		return err
	}

	defer m.Close()
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
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

func WaitForMigrationJob(conn string, logger *slog.Logger, waitOpts ...WaitForMigrationOption) error {
	opts := getWaitForMigrationOptions(waitOpts)

	m, err := getMigrationForConn(conn)
	if err != nil {
		return err
	}
	defer m.Close()

	currentBinaryTargetVersion, err := m.LatestVersion()
	if err != nil {
		return err
	}

	err = retry.Do(
		func() error {
			ver, dirty, err := m.Version()
			if err != nil {
				return retry.Unrecoverable(err)
			}
			if dirty {
				return retry.Unrecoverable(fmt.Errorf("database is dirty, please run migrations manually"))
			}

			logger.Info("waiting for migration job",
				slog.Int("current_db_version", int(ver)),
				slog.Int("current_binary_target_version", int(currentBinaryTargetVersion)),
			)

			if ver >= currentBinaryTargetVersion {
				// We are at least on the version that is required by the binary => good to go
				return nil
			}

			return fmt.Errorf("database is not at the latest version, current version: %d, target version: %d", ver, currentBinaryTargetVersion)
		},
		retry.Delay(opts.Delay),
		retry.MaxDelay(opts.Delay),
		retry.Attempts(opts.MaxRetries),
	)
	if err != nil {
		return fmt.Errorf("failed to wait for migration job: %w", err)
	}

	logger.Info("database migrations are ready")
	return nil
}

func SetMigrationTableName(conn, tableName string) (string, error) {
	parsedURL, err := url.Parse(conn)
	if err != nil {
		return "", err
	}

	values := parsedURL.Query()
	values.Set("x-migrations-table", tableName)
	parsedURL.RawQuery = values.Encode()

	return parsedURL.String(), nil
}
