// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"database/sql"
	"embed"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const (
	MigrationsTable = "schema_om"
)

type Migrate = migrate.Migrate

//go:embed migrations
var OMMigrations embed.FS

type Options struct {
	DB       *sql.DB
	FS       fs.FS
	FSPath   string
	PGConfig *pgx.Config
}

// NewMigrate creates a new migrate instance.
func NewMigrate(opt Options) (*Migrate, error) {
	d, err := iofs.New(opt.FS, opt.FSPath)
	if err != nil {
		return nil, err
	}

	driver, err := pgx.WithInstance(opt.DB, opt.PGConfig)
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("iofs", d, "postgres", driver)
}

func Default(db *sql.DB) (*Migrate, error) {
	return NewMigrate(Options{
		DB:     db,
		FS:     OMMigrations,
		FSPath: "migrations",
		PGConfig: &pgx.Config{
			MigrationsTable: MigrationsTable,
		},
	})
}
