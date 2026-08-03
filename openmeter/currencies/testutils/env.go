package testutils

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciesadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	currenciesservice "github.com/openmeterio/openmeter/openmeter/currencies/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type TestEnv struct {
	Logger     *slog.Logger
	Service    currencies.Service
	Repository currencies.Repository
	Client     *entdb.Client

	db    *testutils.TestDB
	close sync.Once
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		if e.db != nil {
			if err := e.db.EntDriver.Close(); err != nil {
				t.Errorf("failed to close ent driver: %v", err)
			}

			if err := e.db.PGDriver.Close(); err != nil {
				t.Errorf("failed to close postgres driver: %v", err)
			}
		}

		if e.Client != nil {
			if err := e.Client.Close(); err != nil {
				t.Errorf("failed to close ent client: %v", err)
			}
		}
	})
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	logger := testutils.NewDiscardLogger(t)
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	client := db.EntDriver.Client()

	repository, err := currenciesadapter.New(currenciesadapter.Config{
		Client: client,
	})
	require.NoErrorf(t, err, "initializing currencies adapter must not fail")
	require.NotNil(t, repository, "currencies adapter must not be nil")

	service, err := currenciesservice.New(repository)
	require.NoErrorf(t, err, "initializing currencies service must not fail")
	require.NotNil(t, service, "currencies service must not be nil")

	return &TestEnv{
		Logger:     logger,
		Service:    service,
		Repository: repository,
		Client:     client,
		db:         db,
	}
}
