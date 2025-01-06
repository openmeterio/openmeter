package subscriptiontestutils

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

type DBDeps struct {
	DBClient  *db.Client
	EntDriver *entdriver.EntPostgresDriver
	PGDriver  *pgdriver.Driver
}

func (d *DBDeps) Cleanup(t *testing.T) {
	var errs []error

	if err := d.DBClient.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := d.EntDriver.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := d.PGDriver.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Fatalf("failed to cleanup db deps: %v", errors.Join(errs...))
	}
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
		DBClient:  dbClient,
		EntDriver: entDriver,
		PGDriver:  pgDriver,
	}
}
