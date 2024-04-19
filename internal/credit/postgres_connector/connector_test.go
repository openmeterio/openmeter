package postgres_connector

import (
	"log/slog"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/peterldowns/pgtestdb"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	credit_connector "github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_internal "github.com/openmeterio/openmeter/internal/meter"
)

func TestConnector(t *testing.T) {
	meterRepository := meter_internal.NewInMemoryRepository(nil)
	lockManager := inmemory_lock.NewLockManager(time.Second * 10)

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, db_client *db.Client)
	}{
		{
			name:        "ImplementsInterface",
			description: "PostgresConnector implements product.Connector interface",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				assert.Implements(t, (*credit_connector.Connector)(nil), connector)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := entsql.OpenDB(dialect.Postgres, pgtestdb.New(t, pgtestdb.Config{
				DriverName: "pgx",
				User:       "postgres",
				Password:   "postgres",
				Host:       "localhost",
				Port:       "5432",
				Options:    "sslmode=disable",
			}, &EntMigrator{}))
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository, lockManager)
			tc.test(t, connector, databaseClient)
		})
	}
}
