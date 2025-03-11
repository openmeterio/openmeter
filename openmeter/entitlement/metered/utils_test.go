package meteredentitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit"
	credit_postgres_adapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlement_postgresadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	productcatalog_postgresadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type dependencies struct {
	dbClient               *db.Client
	pgDriver               *pgdriver.Driver
	entDriver              *entdriver.EntPostgresDriver
	featureRepo            feature.FeatureRepo
	entitlementRepo        entitlement.EntitlementRepo
	usageResetRepo         meteredentitlement.UsageResetRepo
	grantRepo              grant.Repo
	balanceSnapshotService balance.SnapshotService
	balanceConnector       credit.BalanceConnector
	ownerConnector         grant.OwnerConnector
	streamingConnector     *streamingtestutils.MockStreamingConnector
	creditConnector        credit.CreditConnector
}

// Teardown cleans up the dependencies
func (d *dependencies) Teardown() {
	d.dbClient.Close()
	d.entDriver.Close()
	d.pgDriver.Close()
}

var (
	namespace = "ns1"
	meterSlug = "meter1"
)

// When migrating in parallel with entgo it causes concurrent writes error
var m sync.Mutex

// builds connector with mock streaming and real PG
func setupConnector(t *testing.T) (meteredentitlement.Connector, *dependencies) {
	testLogger := testutils.NewLogger(t)
	tracer := noop.NewTracerProvider().Tracer("test")
	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	meterAdapter, err := meteradapter.New([]meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:         meterSlug,
		Aggregation: meter.MeterAggregationSum,
		// These will be ignored in tests
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

	featureRepo := productcatalog_postgresadapter.NewPostgresFeatureRepo(dbClient, testLogger)
	entitlementRepo := entitlement_postgresadapter.NewPostgresEntitlementRepo(dbClient)
	usageResetRepo := entitlement_postgresadapter.NewPostgresUsageResetRepo(dbClient)
	grantRepo := credit_postgres_adapter.NewPostgresGrantRepo(dbClient)
	balanceSnapshotRepo := credit_postgres_adapter.NewPostgresBalanceSnapshotRepo(dbClient)

	m.Lock()
	defer m.Unlock()
	// migrate db via ent schema upsert
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	mockPublisher := eventbus.NewMock(t)

	// build adapters
	ownerConnector := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterAdapter,
		testLogger,
		tracer,
	)

	transactionManager := enttx.NewCreator(dbClient)

	snapshotService := balance.NewSnapshotService(balance.SnapshotServiceConfig{
		OwnerConnector:     ownerConnector,
		StreamingConnector: streamingConnector,
		Repo:               balanceSnapshotRepo,
	})

	creditConnector := credit.NewCreditConnector(
		credit.CreditConnectorConfig{
			GrantRepo:              grantRepo,
			BalanceSnapshotService: snapshotService,
			OwnerConnector:         ownerConnector,
			StreamingConnector:     streamingConnector,
			Logger:                 testLogger,
			Tracer:                 tracer,
			Granularity:            time.Minute,
			Publisher:              mockPublisher,
			SnapshotGracePeriod:    isodate.MustParse(t, "P1W"),
			TransactionManager:     transactionManager,
		},
	)

	connector := meteredentitlement.NewMeteredEntitlementConnector(
		streamingConnector,
		ownerConnector,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		mockPublisher,
		testLogger,
		tracer,
	)

	return connector, &dependencies{
		dbClient,
		pgDriver,
		entDriver,
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		grantRepo,
		snapshotService,
		creditConnector,
		ownerConnector,
		streamingConnector,
		creditConnector,
	}
}

func assertUsagePeriodEquals(t *testing.T, expected, actual *entitlement.UsagePeriod) {
	assert.NotNil(t, expected, "expected is nil")
	assert.NotNil(t, actual, "actual is nil")
	assert.Equal(t, expected.Interval, actual.Interval, "periods do not match")
	assert.Equal(t, expected.Anchor.Format(time.RFC3339), actual.Anchor.Format(time.RFC3339), "anchors do not match")
}
