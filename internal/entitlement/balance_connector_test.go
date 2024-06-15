package entitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	credit_postgres_adapter "github.com/openmeterio/openmeter/internal/credit/postgresadapter"
	credit_postgres_adapter_db "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlement_postgresadapter "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter"
	entitlement_postgresadapter_db "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	productcatalog_postgresadapter "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter"
	productcatalog_postgresadapter_db "github.com/openmeterio/openmeter/internal/productcatalog/postgresadapter/ent/db"
	streaming_testutils "github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetEntitlementBalance(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := productcatalog.DBCreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		MeterSlug:           meterSlug,
		MeterGroupByFilters: &map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature productcatalog.Feature) entitlement.CreateEntitlementInputs {
		t.Helper()
		return entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			FeatureID:        feature.ID,
			MeasureUsageFrom: testutils.GetRFC3339Time(t, "1024-03-01T00:00:00Z"), // old, override in tests
		}
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies)
	}{
		{
			name: "Should ignore usage before start of measurement",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = startTime
				// create entitlement in db
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(-time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, startTime.Add(time.Hour))
				assert.NoError(t, err)

				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 0.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage if there's no active grant",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, getEntitlement(t, feature))
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				assert.Equal(t, 100.0, entBalance.UsageInPeriod)
				assert.Equal(t, 100.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage until very first grant after reset",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				t.Skip("TODO: Implement test we need reset")
			},
		},
		{
			name: "Should return correct usage and balance",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				// issue grants
				_, err = deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     credit.GrantOwner(entitlement.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				_, err = deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     credit.GrantOwner(entitlement.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: queryTime.Add(time.Hour),
					ExpiresAt:   queryTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				assert.Equal(t, 100.0, entBalance.UsageInPeriod)
				assert.Equal(t, 900.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)
			},
		},
		{
			name: "Should save new snapshot",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(3 * time.Hour) // longer than grace period for saving snapshots

				// issue grants
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(entitlement.ID),
				}

				g1, err := deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    2,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				g2, err := deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour),
					ExpiresAt:   startTime.Add(time.Hour).AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, g1.EffectiveAt.Add(time.Minute*5))
				deps.streaming.AddSimpleEvent(meterSlug, 100, g2.EffectiveAt.Add(time.Minute))

				// add a balance snapshot
				err = deps.balanceSnapshotDB.Save(
					ctx,
					owner, []credit.GrantBalanceSnapshot{
						{
							Balances: credit.GrantBalanceMap{
								g1.ID: 750,
							},
							Overage: 0,
							At:      g1.EffectiveAt.Add(time.Minute),
						},
					})
				assert.NoError(t, err)

				// get last vaild snapshot
				snap1, err := deps.balanceSnapshotDB.GetLatestValidAt(ctx, owner, queryTime)
				assert.NoError(t, err)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 200.0, entBalance.UsageInPeriod) // in total we had 200 usage
				assert.Equal(t, 1550.0, entBalance.Balance)      // 750 + 1000 (g2 amount) - 200 = 1550
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotDB.GetLatestValidAt(ctx, owner, queryTime)
				assert.NoError(t, err)

				// check snapshots
				assert.NotEqual(t, snap1.At, snap2.At)
				assert.Equal(t, 0.0, snap2.Overage)
				assert.Equal(t, credit.GrantBalanceMap{
					g1.ID: 650,  // the grant that existed so far
					g2.ID: 1000, // the grant that was added at this instant
				}, snap2.Balances)
				assert.Equal(t, g2.EffectiveAt, snap2.At)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			connector, deps := setupConnector(t)
			tc.run(t, connector, deps)
		})
	}
}

func TestGetEntitlementHistory(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := productcatalog.DBCreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		MeterSlug:           meterSlug,
		MeterGroupByFilters: &map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature productcatalog.Feature) entitlement.CreateEntitlementInputs {
		t.Helper()
		return entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			FeatureID:        feature.ID,
			MeasureUsageFrom: testutils.GetRFC3339Time(t, "1024-03-01T00:00:00Z"), // old, override in tests
		}
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies)
	}{
		{
			name: "Should return windowed history",
			run: func(t *testing.T, connector entitlement.EntitlementBalanceConnector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				// issue grants
				// grant at start
				_, err = deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant falling on 3h window
				_, err = deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 3),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant inbetween windows
				_, err = deps.grantDB.CreateGrant(ctx, credit.DBCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				windowedHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, entitlement.BalanceHistoryParams{
					From:           startTime,
					To:             queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     entitlement.WindowSizeHour,
				})
				assert.NoError(t, err)

				assert.Len(t, windowedHistory, 12)

				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[0].UsageInPeriod)
				assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
				assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[2].UsageInPeriod)
				assert.Equal(t, 9900.0, windowedHistory[2].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[3].UsageInPeriod)
				assert.Equal(t, 19800.0, windowedHistory[3].BalanceAtStart)
				assert.Equal(t, 19700.0, windowedHistory[4].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[5].UsageInPeriod)
				assert.Equal(t, 19700.0, windowedHistory[5].BalanceAtStart) // even though EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30) grant happens here, it is only recognized at the next window
				assert.Equal(t, 29600.0, windowedHistory[6].BalanceAtStart)
				assert.Equal(t, 29600.0, windowedHistory[7].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				assert.Equal(t, 1100.0, windowedHistory[8].UsageInPeriod)
				assert.Equal(t, 29600.0, windowedHistory[8].BalanceAtStart)
				assert.Equal(t, 28500.0, windowedHistory[9].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))
				assert.Equal(t, 100.0, windowedHistory[11].UsageInPeriod)
				assert.Equal(t, 28500.0, windowedHistory[11].BalanceAtStart)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			connector, deps := setupConnector(t)
			tc.run(t, connector, deps)
		})
	}
}

type testDependencies struct {
	featureDB         productcatalog.FeatureDBConnector
	entitlementDB     entitlement.EntitlementDBConnector
	usageResetDB      entitlement.UsageResetDBConnector
	grantDB           credit.GrantDBConnector
	balanceSnapshotDB credit.BalanceSnapshotDBConnector
	streaming         *streaming_testutils.MockStreamingConnector
}

// builds connector with mock streaming and real PG
func setupConnector(t *testing.T) (entitlement.EntitlementBalanceConnector, *testDependencies) {
	testLogger := testutils.NewLogger(t)

	// TODO: Mock Streaming shouldn't need a highwatermark, thats not a streaming concept
	veryOld, err := time.Parse(time.RFC3339, "1024-03-01T00:00:00Z")
	assert.NoError(t, err)

	streaming := streaming_testutils.NewMockStreamingConnector(t, streaming_testutils.MockStreamingConnectorParams{
		DefaultHighwatermark: veryOld,
	})

	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(t)

	// build db clients
	productcatalogDBClient := productcatalog_postgresadapter_db.NewClient(productcatalog_postgresadapter_db.Driver(driver))
	featureDB := productcatalog_postgresadapter.NewPostgresFeatureDBAdapter(productcatalogDBClient, testLogger)

	entitlementDBClient := entitlement_postgresadapter_db.NewClient(entitlement_postgresadapter_db.Driver(driver))
	entitlementDB := entitlement_postgresadapter.NewPostgresEntitlementDBAdapter(entitlementDBClient)
	usageresetDB := entitlement_postgresadapter.NewPostgresUsageResetDBAdapter(entitlementDBClient)

	grantDbClient := credit_postgres_adapter_db.NewClient(credit_postgres_adapter_db.Driver(driver))
	grantDbConn := credit_postgres_adapter.NewPostgresGrantDBAdapter(grantDbClient)
	balanceSnapshotDbConn := credit_postgres_adapter.NewPostgresBalanceSnapshotDBAdapter(grantDbClient)

	// migrate all clients
	if err := productcatalogDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}
	if err := entitlementDBClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}
	if err := grantDbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	// build adapters
	owner := entitlement.NewEntitlementGrantOwnerAdapter(
		featureDB,
		entitlementDB,
		usageresetDB,
		testLogger,
	)

	balance := credit.NewBalanceConnector(
		grantDbConn,
		balanceSnapshotDbConn,
		owner,
		streaming,
		testLogger,
	)

	connector := entitlement.NewEntitlementBalanceConnector(
		streaming,
		owner,
		balance,
	)

	return connector, &testDependencies{
		featureDB:         featureDB,
		entitlementDB:     entitlementDB,
		usageResetDB:      usageresetDB,
		grantDB:           grantDbConn,
		balanceSnapshotDB: balanceSnapshotDbConn,
		streaming:         streaming,
	}
}
