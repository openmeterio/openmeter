package testutils

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	addonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	addonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	planaddonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/adapter"
	planaddonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type TestEnv struct {
	Logger              *slog.Logger
	Publisher           eventbus.Publisher
	Meter               *meteradapter.TestAdapter
	Feature             feature.FeatureConnector
	Plan                plan.Service
	PlanRepository      plan.Repository
	PlanAddon           planaddon.Service
	PlanAddonRepository planaddon.Repository
	Addon               addon.Service
	AddonRepository     addon.Repository

	Client *entdb.Client
	db     *testutils.TestDB
	close  sync.Once
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	require.NotNilf(t, e.db, "database must be initialized")

	err := e.db.EntDriver.Client().Schema.Create(context.Background())
	require.NoErrorf(t, err, "schema migration must not fail")
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		if e.db != nil {
			if err := e.db.EntDriver.Close(); err != nil {
				t.Errorf("failed to close ent driver: %v", err)
			}

			if err := e.db.PGDriver.Close(); err != nil {
				t.Errorf("failed to postgres driver: %v", err)
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

	// Init logger
	logger := testutils.NewDiscardLogger(t)

	// Init database
	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	// Init event publisher
	publisher := eventbus.NewMock(t)

	// Init meter service
	meterAdapter, err := meteradapter.New(nil)
	require.NoErrorf(t, err, "initializing meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "meter adapter must not be nil")

	// Init feature service
	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(client, logger)
	featureService := feature.NewFeatureConnector(featureAdapter, meterAdapter, publisher, nil)

	// Init plan service
	planAdapter, err := planadapter.New(planadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing plan adapter must not fail")
	require.NotNilf(t, planAdapter, "plan adapter must not be nil")

	planService, err := planservice.New(planservice.Config{
		Adapter:   planAdapter,
		Feature:   featureService,
		Logger:    logger,
		Publisher: publisher,
	})
	require.NoErrorf(t, err, "initializing plan service must not fail")
	require.NotNilf(t, planService, "plan service must not be nil")

	// Init addon service
	addonAdapter, err := addonadapter.New(addonadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing addon adapter must not fail")
	require.NotNilf(t, addonAdapter, "addon adapter must not be nil")

	addonService, err := addonservice.New(addonservice.Config{
		Adapter:   addonAdapter,
		Feature:   featureService,
		Logger:    logger,
		Publisher: publisher,
	})
	require.NoErrorf(t, err, "initializing addon service must not fail")
	require.NotNilf(t, addonService, "addon service must not be nil")

	// Init planaddon service
	planAddonAdapter, err := planaddonadapter.New(planaddonadapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing planaddon adapter must not fail")
	require.NotNilf(t, addonAdapter, "planaddon adapter must not be nil")

	planAddonService, err := planaddonservice.New(planaddonservice.Config{
		Adapter:   planAddonAdapter,
		Plan:      planService,
		Addon:     addonService,
		Logger:    logger,
		Publisher: publisher,
	})
	require.NoErrorf(t, err, "initializing planaddon service must not fail")
	require.NotNilf(t, addonService, "planaddon service must not be nil")

	return &TestEnv{
		Logger:              logger,
		Publisher:           publisher,
		Meter:               meterAdapter,
		Feature:             featureService,
		Plan:                planService,
		PlanRepository:      planAdapter,
		PlanAddon:           planAddonService,
		PlanAddonRepository: planAddonAdapter,
		Addon:               addonService,
		AddonRepository:     addonAdapter,
		db:                  db,
		Client:              client,
	}
}
