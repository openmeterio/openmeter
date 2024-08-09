// A lightweigh migration tool to replace `ent.Schema.Create` calls.
package migrate

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var fs embed.FS

func Up(conn string) error {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, conn)
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
