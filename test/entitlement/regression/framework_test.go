package framework_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	grantrepo "github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	grantdb "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/internal/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	entitlementrepo "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	entitlementdb "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	staticentitlement "github.com/openmeterio/openmeter/internal/entitlement/static"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalogrepo "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter"
	productcatalogdb "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
	streamingtestutils "github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Dependencies struct {
	GrantRepo                credit.GrantRepo
	GrantDB                  *grantdb.Client
	BalanceSnapshotConnector credit.BalanceSnapshotConnector
	GrantConnector           credit.GrantConnector

	EntitlementRepo entitlement.EntitlementRepo
	EntitlementDB   entitlementdb.Client

	EntitlementConnector        entitlement.Connector
	StaticEntitlementConnector  staticentitlement.Connector
	BooleanEntitlementConnector booleanentitlement.Connector
	MeteredEntitlementConnector meteredentitlement.Connector

	Streaming *streamingtestutils.MockStreamingConnector

	FeatureRepo      productcatalog.FeatureRepo
	ProductCatalogDB *productcatalogdb.Client
	FeatureConnector productcatalog.FeatureConnector

	Log *slog.Logger
}

func (d *Dependencies) Close() {
	d.GrantDB.Close()
	d.EntitlementDB.Close()
	d.ProductCatalogDB.Close()
}

func setupDependencies(t *testing.T) Dependencies {
	log := slog.Default()
	ctx := context.Background()
	driver := testutils.InitPostgresDB(t)

	// Init product catalog
	productCatalogDB := productcatalogdb.NewClient(productcatalogdb.Driver(driver))

	if err := productCatalogDB.Schema.Create(ctx); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	featureRepo := productcatalogrepo.NewPostgresFeatureRepo(productCatalogDB, log)

	meters := []models.Meter{
		{
			Namespace:   "namespace-1",
			ID:          "meter-1",
			Slug:        "meter-1",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationCount,
		},
	}

	meterRepo := meter.NewInMemoryRepository(meters)

	featureConnector := productcatalog.NewFeatureConnector(featureRepo, meterRepo) // TODO: meter repo is needed

	// Init grants/credit
	grantDB := grantdb.NewClient(grantdb.Driver(driver))
	if err := grantDB.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	grantRepo := grantrepo.NewPostgresGrantRepo(grantDB)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(grantDB)

	// Init entitlements
	streaming := streamingtestutils.NewMockStreamingConnector(t)

	entitlementDB := entitlementdb.NewClient(entitlementdb.Driver(driver))

	if err := entitlementDB.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(entitlementDB)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(entitlementDB)

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterRepo,
		log,
	)

	balance := credit.NewBalanceConnector(
		grantRepo,
		balanceSnapshotRepo,
		owner,
		streaming,
		log,
	)

	grant := credit.NewGrantConnector(
		owner,
		grantRepo,
		balanceSnapshotRepo,
		time.Minute,
	)

	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		streaming,
		owner,
		balance,
		grant,
		entitlementRepo)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	entitlementConnector := entitlement.NewEntitlementConnector(
		entitlementRepo,
		featureConnector,
		meterRepo,
		meteredEntitlementConnector,
		staticEntitlementConnector,
		booleanEntitlementConnector,
	)

	return Dependencies{
		GrantRepo:      grantRepo,
		GrantDB:        grantDB,
		GrantConnector: grant,

		EntitlementRepo: entitlementRepo,
		EntitlementDB:   *entitlementDB,

		EntitlementConnector:        entitlementConnector,
		StaticEntitlementConnector:  staticEntitlementConnector,
		BooleanEntitlementConnector: booleanEntitlementConnector,
		MeteredEntitlementConnector: meteredEntitlementConnector,

		Streaming: streaming,

		FeatureRepo:      featureRepo,
		ProductCatalogDB: productCatalogDB,
		FeatureConnector: featureConnector,

		Log: log,
	}

}
