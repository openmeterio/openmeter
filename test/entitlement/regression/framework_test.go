package framework_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credithook "github.com/openmeterio/openmeter/openmeter/credit/hook"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	entitlementsubscriptionhook "github.com/openmeterio/openmeter/openmeter/entitlement/hooks/subscription"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	entitlementservice "github.com/openmeterio/openmeter/openmeter/entitlement/service"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	productcatalogrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

type Dependencies struct {
	DBClient  *db.Client
	PGDriver  *pgdriver.Driver
	EntDriver *entdriver.EntPostgresDriver

	GrantRepo              grant.Repo
	BalanceSnapshotService balance.SnapshotService
	GrantConnector         credit.GrantConnector

	EntitlementRepo entitlement.EntitlementRepo

	EntitlementConnector        entitlement.Service
	StaticEntitlementConnector  staticentitlement.Connector
	BooleanEntitlementConnector booleanentitlement.Connector
	MeteredEntitlementConnector meteredentitlement.Connector

	Streaming *streamingtestutils.MockStreamingConnector

	FeatureRepo      feature.FeatureRepo
	FeatureConnector feature.FeatureConnector

	CustomerService customer.Service
	SubjectService  subject.Service

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

	featureConnector := feature.NewFeatureConnector(featureRepo, meterAdapter, eventbus.NewMock(t), nil) // TODO: meter repo is needed

	// Init grants/credit
	grantRepo := grantrepo.NewPostgresGrantRepo(dbClient)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(dbClient)

	// Init entitlements
	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(dbClient)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(dbClient)

	mockPublisher := eventbus.NewMock(t)

	subjectRepo, err := subjectadapter.New(dbClient)
	require.NoError(t, err)

	subjectService, err := subjectservice.New(subjectRepo)
	require.NoError(t, err)

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: log,
	})
	require.NoError(t, err)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: mockPublisher,
	})
	require.NoError(t, err)

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterAdapter,
		customerService,
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
			SnapshotGracePeriod:    datetime.NewISODuration(0, 0, 0, 1, 0, 0, 0),
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

	meteredEntitlementConnector.RegisterHooks(
		meteredentitlement.ConvertHook(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(entitlementsubscriptionhook.EntitlementSubscriptionHookConfig{})),
	)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: log,
	})
	require.NoError(t, err)

	entitlementConnector := entitlementservice.NewEntitlementService(
		entitlementservice.ServiceConfig{
			EntitlementRepo:             entitlementRepo,
			FeatureConnector:            featureConnector,
			CustomerService:             customerService,
			MeterService:                meterAdapter,
			MeteredEntitlementConnector: meteredEntitlementConnector,
			StaticEntitlementConnector:  staticEntitlementConnector,
			BooleanEntitlementConnector: booleanEntitlementConnector,
			Publisher:                   mockPublisher,
			Locker:                      locker,
		},
	)

	entitlementConnector.RegisterHooks(
		entitlementsubscriptionhook.NewEntitlementSubscriptionHook(entitlementsubscriptionhook.EntitlementSubscriptionHookConfig{}),
		credithook.NewEntitlementHook(grantRepo),
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

		CustomerService: customerService,
		SubjectService:  subjectService,

		Log: log,
	}
}

func createCustomerAndSubject(t *testing.T, subjectService subject.Service, customerService customer.Service, ns, key, name string) *customer.Customer {
	t.Helper()
	_, err := subjectService.Create(context.Background(), subject.CreateInput{
		Namespace: ns,
		Key:       key,
	})
	require.NoError(t, err)

	cust, err := customerService.CreateCustomer(context.Background(), customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Key: lo.ToPtr(key),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
			Name: name,
		},
	})
	require.NoError(t, err)

	return cust
}
