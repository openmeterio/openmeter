// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"database/sql"
	"embed"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const (
	MigrationsTable = "schema_om"
)

type Migrate = migrate.Migrate

//go:embed migrations
var OMMigrations embed.FS

type Options struct {
	db       *sql.DB
	fs       fs.FS
	fsPath   string
	pgConfig *postgres.Config
}

// NewMigrate creates a new migrate instance.
func NewMigrate(opt Options) (*Migrate, error) {
	d, err := iofs.New(opt.fs, opt.fsPath)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(opt.db, opt.pgConfig)
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("iofs", d, "postgres", driver)
}

func Default(db *sql.DB) (*Migrate, error) {
	return NewMigrate(Options{
		db:     db,
		fs:     OMMigrations,
		fsPath: "migrations",
		pgConfig: &postgres.Config{
			MigrationsTable: MigrationsTable,
		},
	})
}

func Up(db *sql.DB) error {
	m, err := NewMigrate(Options{
		db:     db,
		fs:     OMMigrations,
		fsPath: "migrations",
		pgConfig: &postgres.Config{
			MigrationsTable: MigrationsTable,
		},
	})
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
