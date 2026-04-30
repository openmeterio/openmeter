package testutils

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// Env is a fully wired app domain test environment backed by a real PostgreSQL database.
type Env struct {
	AppService app.Service
	Adapter    app.Adapter

	Client *entdb.Client
	db     *testutils.TestDB
	close  sync.Once
}

// Close releases all resources held by the environment.
func (e *Env) Close(t *testing.T) {
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

// DBSchemaMigrate runs Ent auto-migration on the test database.
func (e *Env) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	require.NotNilf(t, e.db, "database must be initialized")
	err := e.db.EntDriver.Client().Schema.Create(t.Context())
	require.NoErrorf(t, err, "schema migration must not fail")
}

// NewEnvConfig holds dependencies needed to construct an Env.
type NewEnvConfig struct {
	// BillingService is required to register the Sandbox app factory via appsandbox.NewFactory.
	// When nil and RegisterSandboxFactory is false, the Sandbox listing is not registered.
	BillingService billing.Service

	// RegisterSandboxFactory registers a minimal sandbox factory without requiring a billing service.
	// Use this when tests need GetApp/UpdateApp/UninstallApp to work (which require a registered factory)
	// but don't exercise billing logic.
	RegisterSandboxFactory bool
}

// NewTestEnv creates a fully wired app domain test environment against a real PostgreSQL database.
// If config.BillingService is provided, the Sandbox marketplace listing is pre-registered via
// appsandbox.NewFactory. If config.RegisterSandboxFactory is true and BillingService is nil, a
// minimal no-op sandbox factory is registered instead.
func NewTestEnv(t *testing.T, config NewEnvConfig) *Env {
	t.Helper()

	logger := testutils.NewDiscardLogger(t)
	publisher := eventbus.NewMock(t)

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	appAdapter, err := appadapter.New(appadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing app adapter must not fail")
	require.NotNilf(t, appAdapter, "app adapter must not be nil")

	appSvc, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	require.NoErrorf(t, err, "initializing app service must not fail")
	require.NotNilf(t, appSvc, "app service must not be nil")

	if config.BillingService != nil {
		_, err = appsandbox.NewFactory(t.Context(), appsandbox.Config{
			AppService:     appSvc,
			BillingService: config.BillingService,
		})
		require.NoErrorf(t, err, "registering sandbox marketplace listing must not fail")
	} else if config.RegisterSandboxFactory {
		err = appSvc.RegisterMarketplaceListing(t.Context(), app.RegistryItem{
			Listing: appsandbox.MarketplaceListing,
			Factory: &minimalSandboxFactory{},
		})
		require.NoErrorf(t, err, "registering minimal sandbox factory must not fail")
	}

	return &Env{
		AppService: appSvc,
		Adapter:    appAdapter,
		Client:     client,
		db:         db,
		close:      sync.Once{},
	}
}

// minimalSandboxFactory is a no-op AppFactory for the sandbox listing that does not require a
// billing service. It is used in tests that need GetApp/UpdateApp/UninstallApp to succeed (which
// require a registered factory) but do not exercise billing invoice logic.
type minimalSandboxFactory struct{}

func (f *minimalSandboxFactory) NewApp(_ context.Context, appBase app.AppBase) (app.App, error) {
	return &minimalSandboxApp{AppBase: appBase}, nil
}

func (f *minimalSandboxFactory) UninstallApp(_ context.Context, _ app.UninstallAppInput) error {
	return nil
}

// minimalSandboxApp is a minimal app.App implementation backed only by AppBase.
type minimalSandboxApp struct {
	app.AppBase
}

func (a *minimalSandboxApp) GetEventAppData() (app.EventAppData, error) {
	return app.EventAppData{}, nil
}

func (a *minimalSandboxApp) UpdateAppConfig(_ context.Context, _ app.AppConfigUpdate) error {
	return nil
}

func (a *minimalSandboxApp) GetCustomerData(_ context.Context, _ app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return nil, nil
}

func (a *minimalSandboxApp) UpsertCustomerData(_ context.Context, _ app.UpsertAppInstanceCustomerDataInput) error {
	return nil
}

func (a *minimalSandboxApp) DeleteCustomerData(_ context.Context, _ app.DeleteAppInstanceCustomerDataInput) error {
	return nil
}

// InstallSandboxApp installs a Sandbox app in the given namespace and returns the installed app.
// The Sandbox listing must already be registered (i.e. NewEnvConfig.BillingService was provided).
func InstallSandboxApp(t *testing.T, env *Env, namespace string) app.App {
	t.Helper()

	installedApp, err := env.AppService.InstallMarketplaceListing(t.Context(), app.InstallAppInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeSandbox,
		},
		Namespace: namespace,
		Name:      "Sandbox",
	})
	require.NoErrorf(t, err, "installing sandbox app must not fail")
	require.NotNilf(t, installedApp, "installed sandbox app must not be nil")

	return installedApp
}

// MustGetApp fetches an installed app by ID and fails the test if not found.
func MustGetApp(t *testing.T, env *Env, appID app.AppID) app.App {
	t.Helper()

	gotApp, err := env.AppService.GetApp(t.Context(), appID)
	require.NoErrorf(t, err, "getting app must not fail")
	require.NotNilf(t, gotApp, "app must not be nil")

	return gotApp
}

// NewTestLogger returns a discard logger suitable for tests.
func NewTestLogger(t *testing.T) *slog.Logger {
	return testutils.NewDiscardLogger(t)
}
