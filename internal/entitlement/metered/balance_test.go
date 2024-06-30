package meteredentitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestGetEntitlementBalance(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := productcatalog.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature-1",
		MeterSlug:           &meterSlug,
		MeterGroupByFilters: map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature productcatalog.Feature) entitlement.CreateEntitlementRepoInputs {
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
				Anchor: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				// TODO: properly test these anchors
				Interval: recurrence.RecurrencePeriodYear,
			},
		}

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriod()
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies)
	}{
		{
			name: "Should ignore usage before start of measurement",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// add dummy usage so meter is found
				deps.streaming.AddSimpleEvent(meterSlug, 0, startTime.Add(-time.Minute))

				// reset (empty) entitlement
				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					},
				)
				assert.NoError(t, err)

				// usage on ledger that will be deducted
				deps.streaming.AddSimpleEvent(meterSlug, 600, resetTime.Add(time.Minute))

				// get balance with overage
				queryTime := resetTime.Add(time.Hour)
				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, queryTime)

				assert.NoError(t, err)
				assert.Equal(t, 600.0, entBalance.UsageInPeriod)
				assert.Equal(t, 600.0, entBalance.Overage)
				assert.Equal(t, 0.0, entBalance.Balance)

			},
		},
		{
			name: "Should return correct usage and balance",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				// issue grants
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(entitlement.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(3 * time.Hour) // longer than grace period for saving snapshots

				// issue grants
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(entitlement.ID),
				}

				g1, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    2,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				g2, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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
		{
			name: "Should not save the same snapshot over and over again",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				queryTime := startTime.Add(3 * time.Hour) // longer than grace period for saving snapshots

				// issue grants
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(entitlement.ID),
				}

				g1, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    2,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				g2, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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

				// run the calc again
				entBalance, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 200.0, entBalance.UsageInPeriod) // in total we had 200 usage
				assert.Equal(t, 1550.0, entBalance.Balance)      // 750 + 1000 (g2 amount) - 200 = 1550
				assert.Equal(t, 0.0, entBalance.Overage)

				//FIXME: we shouldn't check things that the contract is unable to tell us
				snaps, err := deps.creditDBCLient.BalanceSnapshot.Query().All(ctx)
				assert.NoError(t, err)
				assert.Len(t, snaps, 2) // one for the initial and one we made last time
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

	exampleFeature := productcatalog.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature1",
		MeterSlug:           &meterSlug,
		MeterGroupByFilters: map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature productcatalog.Feature) entitlement.CreateEntitlementRepoInputs {
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
				Anchor: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				// TODO: properly test these anchors
				Interval: recurrence.RecurrencePeriodYear,
			},
		}

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriod()
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies)
	}{
		{
			name: "Should return windowed history",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
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
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant falling on 3h window
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 3),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant between windows
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					From:           &startTime,
					To:             &queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeHour,
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

				// check returned burndownhistory
				segments := burndownHistory.Segments()
				assert.Len(t, segments, 3)
			},
		},
		{
			name: "If start time is not specified we are defaulting to the last reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// grant at start
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// let's do a reset
				resetTime := startTime.Add(time.Hour * 2)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: true,
					},
				)
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				// grant after the reset
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: resetTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					To:             &queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeHour,
				})
				assert.NoError(t, err)

				assert.Len(t, windowedHistory, 10)

				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[0].UsageInPeriod)
				assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[1].UsageInPeriod)
				assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
				assert.Equal(t, 9800.0, windowedHistory[2].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[3].UsageInPeriod)
				assert.Equal(t, 9800.0, windowedHistory[3].BalanceAtStart) // even though EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30) grant happens here, it is only recognized at the next window
				assert.Equal(t, 9700.0, windowedHistory[4].BalanceAtStart)
				assert.Equal(t, 9700.0, windowedHistory[5].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				assert.Equal(t, 1100.0, windowedHistory[6].UsageInPeriod)
				assert.Equal(t, 9700.0, windowedHistory[6].BalanceAtStart)
				assert.Equal(t, 8600.0, windowedHistory[7].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))
				assert.Equal(t, 100.0, windowedHistory[9].UsageInPeriod)
				assert.Equal(t, 8600.0, windowedHistory[9].BalanceAtStart)

				// check returned burndownhistory
				segments := burndownHistory.Segments()
				assert.Len(t, segments, 2)
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

func TestResetEntitlementUsage(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := productcatalog.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature1",
		MeterSlug:           &meterSlug,
		MeterGroupByFilters: map[string]string{},
	}

	getEntitlement := func(t *testing.T, feature productcatalog.Feature) entitlement.CreateEntitlementRepoInputs {
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
				Anchor: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				// TODO: properly test these anchors
				Interval: recurrence.RecurrencePeriodYear,
			},
		}

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriod()
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies)
	}{
		{
			name: "Should allow resetting usage for the first time with no grants",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				resetTime := startTime.Add(time.Hour * 3)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				startingBalance, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: entitlement.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.NoError(t, err)

				assert.Equal(t, 0.0, startingBalance.UsageInPeriod) // cannot be usage
				assert.Equal(t, 0.0, startingBalance.Balance)       // no balance as there are no grants
				assert.Equal(t, 0.0, startingBalance.Overage)       // cannot be overage
			},
		},
		{
			name: "Should error if requested reset time is before start of measurement",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// resetTime before start of measurement
				resetTime := startTime.Add(-time.Hour)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: entitlement.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.ErrorContains(t, err, "before current usage period start")
			},
		},
		{
			name: "Should error if requested reset time is before current period start",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// save a reset time
				priorResetTime := startTime.Add(time.Hour)
				err = deps.usageResetDB.Save(ctx, meteredentitlement.UsageResetTime{
					NamespacedModel: models.NamespacedModel{Namespace: namespace},
					ResetTime:       priorResetTime,
					EntitlementID:   ent.ID,
				})
				assert.NoError(t, err)

				// resetTime before prior reset time
				resetTime := priorResetTime.Add(-time.Minute)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.ErrorContains(t, err, "before current usage period start")
			},
		},
		{
			name: "Should error if requested reset time is in the future",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				now := time.Now().Truncate(time.Minute)
				aDayAgo := now.Add(-time.Hour * 24)

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &aDayAgo
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streaming.AddSimpleEvent(meterSlug, 100, aDayAgo.Add(time.Minute))

				// resetTime in future
				resetTime := now.Add(time.Minute)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.ErrorContains(t, err, "in the future")
			},
		},
		{
			name: "Should invalidate snapshots after the reset time",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// we force snapshot creation the intended way by checking the balance

				// issue grant
				g1, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				queryTime := startTime.Add(time.Hour * 5) // over grace period
				// we get the balance to force snapshot creation
				_, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, queryTime)
				assert.NoError(t, err)

				// for sanity check that snapshot was created (at g1.EffectiveAt)
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(ent.ID),
				}
				snap, err := deps.balanceSnapshotDB.GetLatestValidAt(ctx, owner, queryTime)
				assert.NoError(t, err)

				assert.Equal(t, g1.EffectiveAt, snap.At)

				// resetTime before snapshot
				resetTime := snap.At.Add(-time.Minute)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
			},
		},
		{
			name: "Should return starting balance after reset with rolled over grant values",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				g1, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 100, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// usage on ledger that will be deducted from g1
				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 500.0, balanceAfterReset.Balance)     // 1000 - 600 = 400 rolled over + MAX(1000 - 0, 100)=100 = 500
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // no overage
				assert.Equal(t, resetTime, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err := deps.creditBalance.GetBalanceOfOwner(ctx, credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(ent.ID),
				}, resetTime)
				assert.NoError(t, err)

				assert.Equal(t, credit.GrantBalanceMap{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Balances)
			},
		},
		{
			name: "Should return proper last reset time after reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, startTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))

				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.NoError(t, err)

				// validate that lastReset time is properly set
				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, resetTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))
			},
		},
		{
			name: "Should return proper last reset time after reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, startTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))

				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.NoError(t, err)

				// validate that lastReset time is properly set
				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, resetTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))
			},
		},
		{
			name: "Should calculate balance for grants taking effect after last saved snapshot",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				g1, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 100, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// usage on ledger that will be deducted from g1
				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// do a reset
				resetTime1 := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime1,
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 500.0, balanceAfterReset.Balance)     // 1000 - 600 = 400 rolled over + MAX(1000 - 0, 100)=100 = 500
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // no overage
				assert.Equal(t, resetTime1, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err := deps.creditBalance.GetBalanceOfOwner(ctx, credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(ent.ID),
				}, resetTime1)
				assert.NoError(t, err)

				assert.Equal(t, credit.GrantBalanceMap{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Balances)

				// issue grants taking effect after first reset
				g3, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      resetTime1.Add(time.Hour * 1),
					ExpiresAt:        resetTime1.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// add usage after reset 1
				deps.streaming.AddSimpleEvent(meterSlug, 300, resetTime1.Add(time.Minute*10))

				// do a 2nd reset
				resetTime2 := resetTime1.Add(time.Hour * 5)
				balanceAfterReset, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime2,
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 1200.0, balanceAfterReset.Balance)    // 1000 + 500 - 300 = 1200
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // no overage
				assert.Equal(t, resetTime2, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err = deps.creditBalance.GetBalanceOfOwner(ctx, credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(ent.ID),
				}, resetTime2)
				assert.NoError(t, err)

				assert.Equal(t, credit.GrantBalanceMap{
					g1.ID: 100,
					g2.ID: 100,
					g3.ID: 1000,
				}, creditBalance.Balances)
			},
		},
		{
			name: "Should properly handle grants issued for the same time as reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// add 0 usage so meter is found in mock
				deps.streaming.AddSimpleEvent(meterSlug, 0, startTime)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				_, err = deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 0, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// do a reset
				resetTime := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				// assert balance after reset is 0 for grant
				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 0.0, balanceAfterReset.Balance)       // 1000 - 1000 = 0

				// issue grants
				g2, err := deps.grantDB.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:          credit.GrantOwner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      resetTime,
					ExpiresAt:        resetTime.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// fetch balance for reset & grant, balance should be full grant amount
				balanceAfterReset, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, resetTime)
				assert.NoError(t, err)

				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, g2.Amount, balanceAfterReset.Balance) // 1000 - 0 = 1000

				// fetch balance for AFTER reset & grant, balance should be full grant amount
				balanceAfterReset, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, resetTime.Add(time.Minute))
				assert.NoError(t, err)

				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, g2.Amount, balanceAfterReset.Balance) // 1000 - 0 = 1000
			},
		},
		{
			name: "Should reseting without anchor update keeps the next reset time intact",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod.Anchor = anchor
				inp.UsagePeriod.Interval = recurrence.RecurrencePeriodDaily
				inp.CurrentUsagePeriod = &recurrence.Period{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: true,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assertUsagePeriodEquals(t, inp.UsagePeriod, ent.UsagePeriod)
			},
		},
		{
			name: "Should reseting with anchor update updates the next reset time too",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod.Interval = recurrence.RecurrencePeriodDaily
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod.Anchor = anchor
				inp.CurrentUsagePeriod = &recurrence.Period{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assertUsagePeriodEquals(t, &entitlement.UsagePeriod{
					Interval: recurrence.RecurrencePeriodDaily,
					Anchor:   resetTime,
				}, ent.UsagePeriod)
			},
		},
		{
			name: "When resetting with anchor update the anchor gets truncated to per minute resolution",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *testDependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureDB.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod.Interval = recurrence.RecurrencePeriodDaily
				inp.UsagePeriod.Anchor = anchor
				inp.CurrentUsagePeriod = &recurrence.Period{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementDB.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streaming.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5).Add(time.Second)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementDB.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assertUsagePeriodEquals(t, &entitlement.UsagePeriod{
					Interval: recurrence.RecurrencePeriodDaily,
					Anchor:   resetTime.Truncate(time.Minute),
				}, ent.UsagePeriod)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			connector, deps := setupConnector(t)
			tc.run(t, connector, deps)
		})
	}
}

type testDependencies struct {
	creditDBCLient    *credit_postgres_adapter_db.Client
	featureDB         productcatalog.FeatureRepo
	entitlementDB     entitlement.EntitlementRepo
	usageResetDB      meteredentitlement.UsageResetRepo
	grantDB           credit.GrantRepo
	balanceSnapshotDB credit.BalanceSnapshotConnector
	creditBalance     credit.BalanceConnector
	streaming         *streaming_testutils.MockStreamingConnector
}

var m sync.Mutex

// builds connector with mock streaming and real PG
func setupConnector(t *testing.T) (meteredentitlement.Connector, *testDependencies) {
	testLogger := testutils.NewLogger(t)

	streaming := streaming_testutils.NewMockStreamingConnector(t)
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
	featureDB := productcatalog_postgresadapter.NewPostgresFeatureRepo(productcatalogDBClient, testLogger)

	entitlementDBClient := entitlement_postgresadapter_db.NewClient(entitlement_postgresadapter_db.Driver(driver))
	entitlementDB := entitlement_postgresadapter.NewPostgresEntitlementRepo(entitlementDBClient)
	usageresetDB := entitlement_postgresadapter.NewPostgresUsageResetRepo(entitlementDBClient)

	grantDbClient := credit_postgres_adapter_db.NewClient(credit_postgres_adapter_db.Driver(driver))
	grantDbConn := credit_postgres_adapter.NewPostgresGrantRepo(grantDbClient)
	balanceSnapshotDbConn := credit_postgres_adapter.NewPostgresBalanceSnapshotRepo(grantDbClient, credit_postgres_adapter.BalanceSnapshotConfig{
		Enabled: true,
	})

	m.Lock()
	defer m.Unlock()
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
	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureDB,
		entitlementDB,
		usageresetDB,
		meterRepo,
		testLogger,
	)

	balance := credit.NewBalanceConnector(
		grantDbConn,
		balanceSnapshotDbConn,
		owner,
		streaming,
		testLogger,
	)

	grant := credit.NewGrantConnector(
		owner,
		grantDbConn,
		balanceSnapshotDbConn,
		time.Minute,
	)

	connector := meteredentitlement.NewMeteredEntitlementConnector(
		streaming,
		owner,
		balance,
		grant,
		entitlementDB,
	)

	return connector, &testDependencies{
		creditDBCLient:    grantDbClient,
		featureDB:         featureDB,
		entitlementDB:     entitlementDB,
		usageResetDB:      usageresetDB,
		grantDB:           grantDbConn,
		balanceSnapshotDB: balanceSnapshotDbConn,
		creditBalance:     balance,
		streaming:         streaming,
	}
}

func assertUsagePeriodEquals(t *testing.T, expected, actual *entitlement.UsagePeriod) {
	assert.NotNil(t, expected, "expected is nil")
	assert.NotNil(t, actual, "actual is nil")
	assert.Equal(t, expected.Interval, actual.Interval, "periods do not match")
	assert.Equal(t, expected.Anchor.Format(time.RFC3339), actual.Anchor.Format(time.RFC3339), "anchors do not match")
}
