package entitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlement_postgresadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalog_postgresadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

// Meant to work for boolean entitlements
type mockTypeConnector struct{}

var _ entitlement.SubTypeConnector = (*mockTypeConnector)(nil)

type mockTypeValue struct{}

func (m *mockTypeValue) HasAccess() bool {
	return true
}

func (m *mockTypeConnector) GetValue(entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	return &mockTypeValue{}, nil
}

func (m *mockTypeConnector) BeforeCreate(ent entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	return &entitlement.CreateEntitlementRepoInputs{
		Namespace:       ent.Namespace,
		FeatureID:       feature.ID,
		FeatureKey:      feature.Key,
		SubjectKey:      ent.SubjectKey,
		EntitlementType: ent.EntitlementType,
		Metadata:        ent.Metadata,
		ActiveFrom:      ent.ActiveFrom,
		ActiveTo:        ent.ActiveTo,
	}, nil
}

func (m *mockTypeConnector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

type dependencies struct {
	dbClient    *db.Client
	pgDriver    *pgdriver.Driver
	entDriver   *entdriver.EntPostgresDriver
	featureRepo feature.FeatureRepo
}

// Teardown cleans up the dependencies
func (d *dependencies) Teardown() {
	d.dbClient.Close()
	d.entDriver.Close()
	d.pgDriver.Close()
}

// When migrating in parallel with entgo it causes concurrent writes error
var m sync.Mutex

func setupDependecies(t *testing.T) (entitlement.Connector, *dependencies) {
	testLogger := testutils.NewLogger(t)

	meterRepo := meter.NewInMemoryRepository([]models.Meter{{
		Slug:        "meter1",
		Namespace:   "ns1",
		Aggregation: models.MeterAggregationMax,
		WindowSize:  models.WindowSizeMinute,
	}})

	// create isolated pg db for tests
	testdb := testutils.InitPostgresDB(t)
	dbClient := testdb.EntDriver.Client()
	pgDriver := testdb.PGDriver
	entDriver := testdb.EntDriver

	featureRepo := productcatalog_postgresadapter.NewPostgresFeatureRepo(dbClient, testLogger)
	entitlementRepo := entitlement_postgresadapter.NewPostgresEntitlementRepo(dbClient)

	m.Lock()
	defer m.Unlock()
	// migrate db
	if err := migrate.Up(testdb.URL); err != nil {
		t.Fatalf("failed to migrate db: %s", err.Error())
	}

	featureConnector := feature.NewFeatureConnector(featureRepo, meterRepo)

	mockPublisher := eventbus.NewMock(t)

	conn := entitlement.NewEntitlementConnector(
		entitlementRepo,
		featureConnector,
		meterRepo,
		&mockTypeConnector{},
		&mockTypeConnector{},
		&mockTypeConnector{},
		mockPublisher,
	)

	deps := &dependencies{
		dbClient:    dbClient,
		pgDriver:    pgDriver,
		entDriver:   entDriver,
		featureRepo: featureRepo,
	}

	return conn, deps
}
