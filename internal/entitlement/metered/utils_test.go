package meteredentitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/balance"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	credit_postgres_adapter "github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	entitlement_postgresadapter "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalog_postgresadapter "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter"
	streaming_testutils "github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type dependencies struct {
	dbClient            *db.Client
	featureRepo         productcatalog.FeatureRepo
	entitlementRepo     entitlement.EntitlementRepo
	usageResetRepo      meteredentitlement.UsageResetRepo
	grantRepo           grant.Repo
	balanceSnapshotRepo balance.SnapshotRepo
	balanceConnector    credit.BalanceConnector
	streamingConnector  *streaming_testutils.MockStreamingConnector
}

// Teardown cleans up the dependencies
func (d *dependencies) Teardown() {
	d.dbClient.Close()
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

	// build db client & adapters
	dbClient := db.NewClient(db.Driver(driver))

	featureRepo := productcatalog_postgresadapter.NewPostgresFeatureRepo(dbClient, testLogger)
	entitlementRepo := entitlement_postgresadapter.NewPostgresEntitlementRepo(dbClient)
	usageResetRepo := entitlement_postgresadapter.NewPostgresUsageResetRepo(dbClient)
	grantRepo := credit_postgres_adapter.NewPostgresGrantRepo(dbClient)
	balanceSnapshotRepo := credit_postgres_adapter.NewPostgresBalanceSnapshotRepo(dbClient)

	m.Lock()
	defer m.Unlock()
	// migrate db
	if err := dbClient.Schema.Create(context.Background()); err != nil {
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

	creditConnector := credit.NewCreditConnector(
		grantRepo,
		balanceSnapshotRepo,
		owner,
		streamingConnector,
		testLogger,
		time.Minute,
		publisher.NewMockTopicPublisher(t),
	)

	connector := meteredentitlement.NewMeteredEntitlementConnector(
		streamingConnector,
		owner,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		publisher.NewMockTopicPublisher(t),
	)

	return connector, &dependencies{
		dbClient,
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		grantRepo,
		balanceSnapshotRepo,
		creditConnector,
		streamingConnector,
	}
}

func assertUsagePeriodEquals(t *testing.T, expected, actual *entitlement.UsagePeriod) {
	assert.NotNil(t, expected, "expected is nil")
	assert.NotNil(t, actual, "actual is nil")
	assert.Equal(t, expected.Interval, actual.Interval, "periods do not match")
	assert.Equal(t, expected.Anchor.Format(time.RFC3339), actual.Anchor.Format(time.RFC3339), "anchors do not match")
}
