package subscriptiontestutils

import (
	"context"
	"sync"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

type DBDeps struct {
	dbClient  *db.Client
	entDriver *entdriver.EntPostgresDriver
	pgDriver  *pgdriver.Driver
}

func (d *DBDeps) Cleanup() {
	d.dbClient.Close()
	d.entDriver.Close()
	d.pgDriver.Close()
}

var m sync.Mutex

func SetupDBDeps(t *testing.T) *DBDeps {
	t.Helper()

	m.Lock()
	defer m.Unlock()

	testdb := testutils.InitPostgresDB(t)
	dbClient := testdb.EntDriver.Client()
	pgDriver := testdb.PGDriver
	entDriver := testdb.EntDriver

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return &DBDeps{
		dbClient:  dbClient,
		entDriver: entDriver,
		pgDriver:  pgDriver,
	}
}
