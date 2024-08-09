// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"embed"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type Migrate = migrate.Migrate

//go:embed migrations
var omMigrations embed.FS

// NewMigrate creates a new migrate instance.
// fs is expected to contain a migrations directory with the migration files.
func NewMigrate(conn string, fs fs.FS) (*Migrate, error) {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", d, conn)
}

func Up(conn string) error {
	m, err := NewMigrate(conn, omMigrations)
	if err != nil {
		return err
	}

	defer m.Close()
	err = m.Up()
	if err != nil {
		return err
	}
	return nil
}
