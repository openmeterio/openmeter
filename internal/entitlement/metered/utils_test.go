package meteredentitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	credit_postgres_adapter "github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	credit_postgres_adapter_db "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	entitlement_postgresadapter "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	entitlement_postgresadapter_db "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalog_postgresadapter "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter"
	productcatalog_postgresadapter_db "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
	streaming_testutils "github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/assert"
)

type dependencies struct {
	creditDBClient         *credit_postgres_adapter_db.Client
	productcatalogDBClient *productcatalog_postgresadapter_db.Client
	entitlementDBClient    *entitlement_postgresadapter_db.Client
	featureRepo            productcatalog.FeatureRepo
	entitlementRepo        entitlement.EntitlementRepo
	usageResetRepo         meteredentitlement.UsageResetRepo
	grantRepo              credit.GrantRepo
	balanceSnapshotRepo    credit.BalanceSnapshotRepo
	balanceConnector       credit.BalanceConnector
	streamingConnector     *streaming_testutils.MockStreamingConnector
}

// Teardown cleans up the dependencies
func (d *dependencies) Teardown() {
	d.creditDBClient.Close()
	d.productcatalogDBClient.Close()
	d.entitlementDBClient.Close()
}

// When migrating in parallel with entgo it causes concurrent writes error
var m sync.Mutex

// builds connector with mock streaming and real PG
func setupConnector(t *testing.T) (meteredentitlement.Connector, *dependencies) {
	testLogger := testutils.NewLogger(t)

	streamingConnector := streaming_testutils.NewMockStreamingConnector(t)
	meterRepo := meter.NewInMemoryRepository([]models.Meter{{
		Slug:        "meter1",
		Namespace:   "ns1",
		Aggregation: models.MeterAggregationMax,
		WindowSize:  models.WindowSizeMinute,
	}})

	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(t)

	// build db clients
	productcatalogDBClient := productcatalog_postgresadapter_db.NewClient(productcatalog_postgresadapter_db.Driver(driver))
	featureRepo := productcatalog_postgresadapter.NewPostgresFeatureRepo(productcatalogDBClient, testLogger)

	entitlementDBClient := entitlement_postgresadapter_db.NewClient(entitlement_postgresadapter_db.Driver(driver))
	entitlementRepo := entitlement_postgresadapter.NewPostgresEntitlementRepo(entitlementDBClient)
	usageResetRepo := entitlement_postgresadapter.NewPostgresUsageResetRepo(entitlementDBClient)

	creditDBClient := credit_postgres_adapter_db.NewClient(credit_postgres_adapter_db.Driver(driver))
	grantRepo := credit_postgres_adapter.NewPostgresGrantRepo(creditDBClient)
	balanceSnapshotRepo := credit_postgres_adapter.NewPostgresBalanceSnapshotRepo(creditDBClient)

	m.Lock()
	defer m.Unlock()
	// migrate all clients
	if err := productcatalogDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}
	if err := entitlementDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}
	if err := creditDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	// build adapters
	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterRepo,
		testLogger,
	)

	balanceConnector := credit.NewBalanceConnector(
		grantRepo,
		balanceSnapshotRepo,
		owner,
		streamingConnector,
		testLogger,
	)

	grant := credit.NewGrantConnector(
		owner,
		grantRepo,
		balanceSnapshotRepo,
		time.Minute,
	)

	connector := meteredentitlement.NewMeteredEntitlementConnector(
		streamingConnector,
		owner,
		balanceConnector,
		grant,
		entitlementRepo,
	)

	return connector, &dependencies{
		creditDBClient,
		productcatalogDBClient,
		entitlementDBClient,
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		grantRepo,
		balanceSnapshotRepo,
		balanceConnector,
		streamingConnector,
	}
}

func assertUsagePeriodEquals(t *testing.T, expected, actual *entitlement.UsagePeriod) {
	assert.NotNil(t, expected, "expected is nil")
	assert.NotNil(t, actual, "actual is nil")
	assert.Equal(t, expected.Interval, actual.Interval, "periods do not match")
	assert.Equal(t, expected.Anchor.Format(time.RFC3339), actual.Anchor.Format(time.RFC3339), "anchors do not match")
}
