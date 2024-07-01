package framework_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestScenario(t *testing.T) {
	defer clock.ResetTime()
	log := slog.Default()
	ctx := context.Background()
	driver := testutils.InitPostgresDB(t)

	// Init product catalog
	productCatalogDB := productcatalogdb.NewClient(productcatalogdb.Driver(driver))
	defer productCatalogDB.Close()

	if err := productCatalogDB.Schema.Create(context.Background()); err != nil {
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

	assert := assert.New(t)
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
	defer entitlementDB.Close()

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

	entitlementConnector := entitlement.NewEntitlementConnector(
		entitlementRepo,
		featureConnector,
		meterRepo,
		meteredEntitlementConnector,
		staticentitlement.NewStaticEntitlementConnector(),
		booleanentitlement.NewBooleanEntitlementConnector(),
	)
	// Let's create a feature

	feature, err := featureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// Let's create a new entitlement for the feature

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:21Z"))
	entitlement, err := entitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature.ID,
		FeatureKey:      &feature.Key,
		SubjectKey:      "subject-1",
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-06-28T14:48:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's grant some credit

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:24Z"))
	grant1, err := grant.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      10,
			Priority:    5,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant1)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:36:33Z"))
	grant2, err := grant.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      20,
			Priority:    3,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:36:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationDay,
			},
			ResetMaxRollover: 20,
		})
	assert.NoError(err)
	assert.NotNil(grant2)

	// Hack: this is in the future, but at least it won't return an error
	streaming.AddSimpleEvent("meter-1", 1, testutils.GetRFC3339Time(t, "2025-06-28T14:36:00Z"))

	// Let's query the usage
	currentBalance, err := meteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-06-28T14:36:45Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(30.0, currentBalance.Balance)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:30:41Z"))
	// Let's query the usage
	currentBalance, err = meteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-06-28T14:30:41Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(10.0, currentBalance.Balance)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:35:54Z"))
	grant3, err := grant.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      100,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T15:39:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant3)

	// There should be a snapshot created
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:37:18Z"))
	reset, err := meteredEntitlementConnector.ResetEntitlementUsage(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		meteredentitlement.ResetEntitlementUsageParams{
			At:           testutils.GetRFC3339Time(t, "2024-06-29T14:36:00Z"),
			RetainAnchor: false,
		},
	)
	assert.NoError(err)
	assert.NotNil(reset)

	now := clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:42:41Z"))
	// Let's query the usage
	currentBalance, err = meteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		now)
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(0.0, currentBalance.Balance)
}
