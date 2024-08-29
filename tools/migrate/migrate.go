// A very lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"embed"
	"io/fs"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const (
	MigrationsTable = "schema_om"
)

type Migrate = migrate.Migrate

//go:embed migrations
var OMMigrations embed.FS

// NewMigrate creates a new migrate instance.
func NewMigrate(conn string, fs fs.FS, fsPath string) (*Migrate, error) {
	d, err := iofs.New(fs, fsPath)
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", d, conn)
}

func Up(conn string) error {
	conn, err := SetMigrationTableName(conn, MigrationsTable)
	if err != nil {
		return err
	}
	m, err := NewMigrate(conn, OMMigrations, "migrations")
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
