package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Meant to work for boolean entitlements
type mockTypeConnector struct{}

var _ entitlement.SubTypeConnector = (*mockTypeConnector)(nil)

type mockTypeValue struct{}

func (m *mockTypeValue) HasAccess() bool {
	return true
}

func (m *mockTypeConnector) GetValue(ctx context.Context, entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
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
	dbClient           *db.Client
	pgDriver           *pgdriver.Driver
	entDriver          *entdriver.EntPostgresDriver
	featureRepo        feature.FeatureRepo
	streamingConnector *streamingtestutils.MockStreamingConnector
	subjectService     subject.Service
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

	meterAdapter, err := meteradapter.New([]meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: "ns1",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:           "meter1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	}})
	if err != nil {
		t.Fatalf("failed to create meter adapter: %v", err)
	}

	// create isolated pg db for tests
	testdb := testutils.InitPostgresDB(t)
	dbClient := testdb.EntDriver.Client()
	pgDriver := testdb.PGDriver
	entDriver := testdb.EntDriver

	m.Lock()
	defer m.Unlock()

	// migrate db via ent schema upsert
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: testLogger,
	})
	require.NoError(t, err)

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbClient,
		StreamingConnector: streamingConnector,
		Logger:             testLogger,
		Tracer:             noop.NewTracerProvider().Tracer("test"),
		MeterService:       meterAdapter,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	// Create subject adapter and service
	subjectAdapter, err := subjectadapter.New(dbClient)
	if err != nil {
		t.Fatalf("failed to create subject adapter: %v", err)
	}

	subjectService, err := subjectservice.New(subjectAdapter)
	if err != nil {
		t.Fatalf("failed to create subject service: %v", err)
	}

	deps := &dependencies{
		dbClient:           dbClient,
		pgDriver:           pgDriver,
		entDriver:          entDriver,
		featureRepo:        entitlementRegistry.FeatureRepo,
		streamingConnector: streamingConnector,
		subjectService:     subjectService,
	}

	return entitlementRegistry.Entitlement, deps
}
