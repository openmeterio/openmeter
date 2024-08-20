package entutils

import (
	"database/sql"

	"entgo.io/ent/dialect"
	entDialectSQL "entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/internal/ent/db"
)

type EntPostgresDriver struct {
	driver *entDialectSQL.Driver
	client *entdb.Client
}

// Close releases all the underlying resources.
func (d *EntPostgresDriver) Close() error {
	if err := d.client.Close(); err != nil {
		return err
	}

	if err := d.driver.Close(); err != nil {
		return err
	}

	return nil
}

// Driver returns the underlying Driver.
func (d *EntPostgresDriver) Driver() *entDialectSQL.Driver {
	return d.driver
}

// Client returns the underlying Client.
func (d *EntPostgresDriver) Client() *entdb.Client {
	return d.client
}

// Clone returns a new EntPostgresDriver initialized by using the underlying *sql.DB.
func (d *EntPostgresDriver) Clone() *EntPostgresDriver {
	driver := entDialectSQL.OpenDB(dialect.Postgres, d.driver.DB())
	client := entdb.NewClient(entdb.Driver(driver))

	return &EntPostgresDriver{
		driver: driver,
		client: client,
	}
}

// NewEntPostgresDriver returns a EntPostgresDriver created from *sql.DB.
func NewEntPostgresDriver(db *sql.DB) *EntPostgresDriver {
	driver := entDialectSQL.OpenDB(dialect.Postgres, db)
	client := entdb.NewClient(entdb.Driver(driver))

	return &EntPostgresDriver{
		driver: driver,
		client: client,
	}
}
