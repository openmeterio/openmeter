package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
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
		Namespace:        ent.Namespace,
		FeatureID:        feature.ID,
		FeatureKey:       feature.Key,
		UsageAttribution: ent.UsageAttribution,
		EntitlementType:  ent.EntitlementType,
		Metadata:         ent.Metadata,
		ActiveFrom:       ent.ActiveFrom,
		ActiveTo:         ent.ActiveTo,
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
	customerService    customer.Service
	meterService       meter.ManageService
	registry           *registry.Entitlement
}

// Teardown cleans up the dependencies
func (d *dependencies) Teardown() {
	d.dbClient.Close()
	d.entDriver.Close()
	d.pgDriver.Close()
}

// When migrating in parallel with entgo it causes concurrent writes error
var m sync.Mutex

func setupDependecies(t *testing.T) (entitlement.Service, *dependencies) {
	testLogger := testutils.NewLogger(t)

	meterService, err := meteradapter.NewManage(nil)
	if err != nil {
		t.Fatalf("failed to create meter service: %v", err)
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
		MeterService:       meterService,
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

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: testLogger,
	})
	if err != nil {
		t.Fatalf("failed to create customer adapter: %v", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: eventbus.NewMock(t),
	})
	if err != nil {
		t.Fatalf("failed to create customer service: %v", err)
	}

	deps := &dependencies{
		dbClient:           dbClient,
		pgDriver:           pgDriver,
		entDriver:          entDriver,
		featureRepo:        entitlementRegistry.FeatureRepo,
		streamingConnector: streamingConnector,
		subjectService:     subjectService,
		customerService:    customerService,
		meterService:       meterService,
		registry:           entitlementRegistry,
	}

	return entitlementRegistry.Entitlement, deps
}

func createCustomerAndSubject(t *testing.T, subjectService subject.Service, customerService customer.Service, ns, key, name string) *customer.Customer {
	t.Helper()

	_, err := subjectService.Create(t.Context(), subject.CreateInput{
		Namespace: ns,
		Key:       key,
	})
	require.NoError(t, err)

	cust, err := customerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(key),
			Name: name,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		},
	})
	require.NoError(t, err)

	return cust
}
