package postgres_connector

import (
	"log/slog"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_internal "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/testutils"
)

func TestConnector(t *testing.T) {
	meterRepository := meter_internal.NewInMemoryRepository(nil)

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, db_client *db.Client)
	}{
		{
			name:        "ImplementsInterface",
			description: "PostgresConnector implements feature.Connector interface",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				assert.Implements(t, (*credit.Connector)(nil), connector)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := testutils.InitPostgresDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository, PostgresConnectorConfig{
				WindowSize: time.Minute,
			})
			tc.test(t, connector, databaseClient)
		})
	}
}
