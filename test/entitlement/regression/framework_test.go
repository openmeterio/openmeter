package framework_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	productcatalogrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Dependencies struct {
	DBClient  *db.Client
	PGDriver  *pgdriver.Driver
	EntDriver *entdriver.EntPostgresDriver

	GrantRepo              grant.Repo
	BalanceSnapshotService balance.SnapshotService
	GrantConnector         credit.GrantConnector

	EntitlementRepo entitlement.EntitlementRepo

	EntitlementConnector        entitlement.Connector
	StaticEntitlementConnector  staticentitlement.Connector
	BooleanEntitlementConnector booleanentitlement.Connector
	MeteredEntitlementConnector meteredentitlement.Connector

	Streaming *streamingtestutils.MockStreamingConnector

	FeatureRepo      feature.FeatureRepo
	FeatureConnector feature.FeatureConnector

	Log *slog.Logger
}

func (d *Dependencies) Close() {
	d.DBClient.Close()
	d.EntDriver.Close()
	d.PGDriver.Close()
}

func setupDependencies(t *testing.T) Dependencies {
	log := slog.Default()
	driver := testutils.InitPostgresDB(t)
	// init db
	dbClient := db.NewClient(db.Driver(driver.EntDriver.Driver()))
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	tracer := noop.NewTracerProvider().Tracer("test")

	// Init product catalog
	featureRepo := productcatalogrepo.NewPostgresFeatureRepo(dbClient, log)

	meters := []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: "namespace-1",
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:         "meter-1",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "test",
		},
	}

	streaming := streamingtestutils.NewMockStreamingConnector(t)

	meterAdapter, err := meteradapter.New(meters)
	if err != nil {
		t.Fatalf("failed to create meter adapter: %v", err)
	}

	featureConnector := feature.NewFeatureConnector(featureRepo, meterAdapter) // TODO: meter repo is needed

	// Init grants/credit
	grantRepo := grantrepo.NewPostgresGrantRepo(dbClient)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(dbClient)

	// Init entitlements
	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(dbClient)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(dbClient)

	mockPublisher := eventbus.NewMock(t)

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterAdapter,
		log,
		tracer,
	)

	balanceSnapshotService := balance.NewSnapshotService(balance.SnapshotServiceConfig{
		OwnerConnector:     owner,
		StreamingConnector: streaming,
		Repo:               balanceSnapshotRepo,
	})

	transactionManager := enttx.NewCreator(dbClient)

	creditConnector := credit.NewCreditConnector(
		credit.CreditConnectorConfig{
			GrantRepo:              grantRepo,
			BalanceSnapshotService: balanceSnapshotService,
			OwnerConnector:         owner,
			StreamingConnector:     streaming,
			Logger:                 log,
			Tracer:                 tracer,
			Granularity:            time.Minute,
			Publisher:              mockPublisher,
			TransactionManager:     transactionManager,
			SnapshotGracePeriod:    isodate.NewPeriod(0, 0, 0, 1, 0, 0, 0),
		},
	)

	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		streaming,
		owner,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		mockPublisher,
		log,
		tracer,
	)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	entitlementConnector := entitlement.NewEntitlementConnector(
		entitlementRepo,
		featureConnector,
		meterAdapter,
		meteredEntitlementConnector,
		staticEntitlementConnector,
		booleanEntitlementConnector,
		mockPublisher,
	)

	return Dependencies{
		DBClient:  dbClient,
		PGDriver:  driver.PGDriver,
		EntDriver: driver.EntDriver,

		GrantRepo:      grantRepo,
		GrantConnector: creditConnector,

		EntitlementRepo: entitlementRepo,

		EntitlementConnector:        entitlementConnector,
		StaticEntitlementConnector:  staticEntitlementConnector,
		BooleanEntitlementConnector: booleanEntitlementConnector,
		MeteredEntitlementConnector: meteredEntitlementConnector,

		BalanceSnapshotService: balanceSnapshotService,

		Streaming: streaming,

		FeatureRepo:      featureRepo,
		FeatureConnector: featureConnector,

		Log: log,
	}
}
