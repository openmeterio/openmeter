package meteredentitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit/balance"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

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

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriodAt(time.Now()) // This should be calculated properly when testing batch resets
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should allow resetting usage for the first time with no grants",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				resetTime := startTime.Add(time.Hour * 3)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// save a reset time
				priorResetTime := startTime.Add(time.Hour)
				err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetTime{
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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				now := time.Now().Truncate(time.Minute)
				aDayAgo := now.Add(-time.Hour * 24)

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &aDayAgo
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, aDayAgo.Add(time.Minute))

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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// we force snapshot creation the intended way by checking the balance

				// issue grant
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     grant.Owner(ent.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				queryTime := startTime.Add(time.Hour * 5) // over grace period
				// we get the balance to force snapshot creation
				_, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, queryTime)
				assert.NoError(t, err)

				// for sanity check that snapshot was created (at g1.EffectiveAt)
				owner := grant.NamespacedOwner{
					Namespace: namespace,
					ID:        grant.Owner(ent.ID),
				}
				snap, err := deps.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, queryTime)
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
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 100, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// usage on ledger that will be deducted from g1
				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

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
				creditBalance, err := deps.balanceConnector.GetBalanceOfOwner(ctx, grant.NamespacedOwner{
					Namespace: namespace,
					ID:        grant.Owner(ent.ID),
				}, resetTime)
				assert.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Balances)
			},
		},
		{
			name: "Should return proper last reset time after reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, startTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.NoError(t, err)

				// validate that lastReset time is properly set
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assert.Equal(t, resetTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))
			},
		},
		{
			name: "Should calculate balance for grants taking effect after last saved snapshot",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        startTime.AddDate(0, 0, 3),
					ResetMaxRollover: 100, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// usage on ledger that will be deducted from g1
				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

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
				creditBalance, err := deps.balanceConnector.GetBalanceOfOwner(ctx, grant.NamespacedOwner{
					Namespace: namespace,
					ID:        grant.Owner(ent.ID),
				}, resetTime1)
				assert.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Balances)

				// issue grants taking effect after first reset
				g3, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      resetTime1.Add(time.Hour * 1),
					ExpiresAt:        resetTime1.AddDate(0, 0, 3),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// add usage after reset 1
				deps.streamingConnector.AddSimpleEvent(meterSlug, 300, resetTime1.Add(time.Minute*10))

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
				creditBalance, err = deps.balanceConnector.GetBalanceOfOwner(ctx, grant.NamespacedOwner{
					Namespace: namespace,
					ID:        grant.Owner(ent.ID),
				}, resetTime2)
				assert.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 100,
					g2.ID: 100,
					g3.ID: 1000,
				}, creditBalance.Balances)
			},
		},
		{
			name: "Should properly handle grants issued for the same time as reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// add 0 usage so meter is found in mock
				deps.streamingConnector.AddSimpleEvent(meterSlug, 0, startTime)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
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
				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
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
			name: "Should properly handle grants expiring the same time as reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")
				resetTime := startTime.AddDate(0, 0, 3)

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// add 0 usage so meter is found in mock
				deps.streamingConnector.AddSimpleEvent(meterSlug, 0, startTime)

				// create entitlement in db
				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          grant.Owner(ent.ID),
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        resetTime,
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				// do a reset
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				// assert balance after reset is 0 for grant
				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 0.0, balanceAfterReset.Balance)       // Grant expires at reset time so we should see no balance
			},
		},
		{
			name: "Should reseting without anchor update keeps the next reset time intact",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
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

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: true,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assertUsagePeriodEquals(t, inp.UsagePeriod, ent.UsagePeriod)
			},
		},
		{
			name: "Should reseting with anchor update updates the next reset time too",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
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

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				assert.NoError(t, err)
				assertUsagePeriodEquals(t, &entitlement.UsagePeriod{
					Interval: recurrence.RecurrencePeriodDaily,
					Anchor:   resetTime,
				}, ent.UsagePeriod)
			},
		},
		{
			name: "When resetting with anchor update the anchor gets truncated to per minute resolution",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
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

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5).Add(time.Second)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				assert.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
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
			defer deps.Teardown()
			tc.run(t, connector, deps)
		})
	}
}
