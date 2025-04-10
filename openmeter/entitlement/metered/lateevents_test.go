package meteredentitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit"
	credit_postgres_adapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// To test late events well add events before and after execution
type inconsistentCreditConnector struct {
	credit.CreditConnector
	AddSimpleEvent func(meterSlug string, value float64, at time.Time)
}

func (c *inconsistentCreditConnector) GetBalanceAt(ctx context.Context, ownerID models.NamespacedID, at time.Time) (engine.RunResult, error) {
	relevantTime := at.Add(-time.Minute)

	c.AddSimpleEvent(meterSlug, 5, relevantTime)
	res, err := c.CreditConnector.GetBalanceAt(ctx, ownerID, at)
	c.AddSimpleEvent(meterSlug, 5, relevantTime)

	return res, err
}

func (c *inconsistentCreditConnector) GetBalanceForPeriod(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (engine.RunResult, error) {
	relevantTime := period.To.Add(-time.Minute)

	c.AddSimpleEvent(meterSlug, 5, relevantTime)
	res, err := c.CreditConnector.GetBalanceForPeriod(ctx, ownerID, period)
	c.AddSimpleEvent(meterSlug, 5, relevantTime)

	return res, err
}

func TestGetEntitlementBalanceConsistency(t *testing.T) {
	exampleFeature := feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature-1",
		MeterSlug:           &meterSlug,
		MeterGroupByFilters: map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature feature.Feature) entitlement.CreateEntitlementRepoInputs {
		t.Helper()
		input := entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        feature.ID,
			FeatureKey:       feature.Key,
			SubjectKey:       "subject1",
			MeasureUsageFrom: convert.ToPointer(testutils.GetRFC3339Time(t, "1024-03-01T00:00:00Z")), // old, override in tests
			EntitlementType:  entitlement.EntitlementTypeMetered,
			IssueAfterReset:  convert.ToPointer(0.0),
			IsSoftLimit:      convert.ToPointer(false),
			UsagePeriod: &entitlement.UsagePeriod{
				Anchor: getAnchor(t),
				// Yearly interval is used which helps adjust to the correct period
				Interval: timeutil.RecurrencePeriodYear,
			},
		}

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriodAt(time.Now())
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	setupMockedConnector := func(t *testing.T) (meteredentitlement.Connector, *dependencies) {
		testLogger := testutils.NewLogger(t)
		tracer := noop.NewTracerProvider().Tracer("test")

		streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
		meterAdapter, err := meteradapter.New([]meter.Meter{{
			Key: meterSlug,
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ID:   "managed-resource-1",
				Name: "managed-resource-1",
				ManagedModel: models.ManagedModel{
					CreatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					UpdatedAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				},
			},
			Aggregation: meter.MeterAggregationSum,
			WindowSize:  meter.WindowSizeMinute,
			// These will be ignored in tests
			EventType:     "test",
			ValueProperty: convert.ToPointer("$.value"),
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

		balanceSnapshotService := balance.NewSnapshotService(balance.SnapshotServiceConfig{
			OwnerConnector:     ownerConnector,
			StreamingConnector: streamingConnector,
			Repo:               balanceSnapshotRepo,
		})

		transactionManager := enttx.NewCreator(dbClient)

		creditConnector := credit.NewCreditConnector(
			credit.CreditConnectorConfig{
				GrantRepo:              grantRepo,
				BalanceSnapshotService: balanceSnapshotService,
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

		inconsistentCreditConnector := &inconsistentCreditConnector{
			CreditConnector: creditConnector,
			AddSimpleEvent:  streamingConnector.AddSimpleEvent,
		}

		connector := meteredentitlement.NewMeteredEntitlementConnector(
			streamingConnector,
			ownerConnector,
			inconsistentCreditConnector,
			inconsistentCreditConnector,
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
			balanceSnapshotService,
			inconsistentCreditConnector,
			ownerConnector,
			streamingConnector,
			inconsistentCreditConnector,
		}
	}

	t.Run("Should return consistent balance and usage values if there are late events", func(t *testing.T) {
		connector, deps := setupMockedConnector(t)
		defer deps.Teardown()

		ctx := context.Background()
		startTime := getAnchor(t)
		clock.SetTime(startTime)
		defer clock.ResetTime()

		// create featute in db
		feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
		assert.NoError(t, err)

		// create entitlement in db
		inp := getEntitlement(t, feature)
		inp.MeasureUsageFrom = &startTime
		inp.UsagePeriod.Interval = timeutil.RecurrencePeriodMonth
		entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
		assert.NoError(t, err)

		queryTime := startTime.AddDate(0, 0, 10) // longer than grace period for saving snapshots

		g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
			OwnerID:          entitlement.ID,
			Namespace:        namespace,
			Amount:           1000,
			ResetMaxRollover: 1000,
			Priority:         2,
			EffectiveAt:      startTime,
			ExpiresAt:        startTime.AddDate(0, 5, 0),
		})
		assert.NoError(t, err)

		// register usage for meter & feature
		deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute*5))

		clock.SetTime(queryTime)

		entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
		assert.NoError(t, err)

		// Let's validate that balance usage and overage adds up to the grant amount
		assert.Equal(t, g1.Amount, entBalance.UsageInPeriod+entBalance.Balance-entBalance.Overage)
	})
}
