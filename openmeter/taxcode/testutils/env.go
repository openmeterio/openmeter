package testutils

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
	taxcodeservice "github.com/openmeterio/openmeter/openmeter/taxcode/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type TestEnv struct {
	Logger  *slog.Logger
	Service taxcode.Service
	Client  *entdb.Client
	db      *testutils.TestDB
	close   sync.Once
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	require.NotNilf(t, e.db, "database must be initialized")

	err := e.db.EntDriver.Client().Schema.Create(t.Context())
	require.NoErrorf(t, err, "schema migration must not fail")
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		if e.Client != nil {
			if err := e.Client.Close(); err != nil {
				t.Errorf("failed to close ent client: %v", err)
			}
		}

		if e.db != nil {
			if err := e.db.EntDriver.Close(); err != nil {
				t.Errorf("failed to close ent driver: %v", err)
			}

			if err := e.db.PGDriver.Close(); err != nil {
				t.Errorf("failed to close postgres driver: %v", err)
			}
		}
	})
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	logger := testutils.NewDiscardLogger(t)

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	env := &TestEnv{
		Logger: logger,
		Client: client,
		db:     db,
	}
	t.Cleanup(func() { env.Close(t) })

	adapter, err := taxcodeadapter.New(taxcodeadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing taxcode adapter must not fail")

	orgDefaultsAdapter, err := taxcodeadapter.NewOrganizationDefaultTaxCodesAdapter(taxcodeadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing organization defaults adapter must not fail")

	svc, err := taxcodeservice.New(taxcodeservice.Config{
		Adapter:                     adapter,
		OrganizationDefaultsAdapter: orgDefaultsAdapter,
		Logger:                      logger,
	})
	require.NoErrorf(t, err, "initializing taxcode service must not fail")

	env.Service = svc

	return env
}
