package meteredentitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestResetEntitlementUsage(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature1",
		MeterSlug:           &meterSlug,
		MeterGroupByFilters: map[string]filter.FilterString{},
	}

	getEntitlement := func(t *testing.T, feature feature.Feature, usageAttribution streaming.CustomerUsageAttribution) entitlement.CreateEntitlementRepoInputs {
		t.Helper()
		input := entitlement.CreateEntitlementRepoInputs{
			Namespace:        namespace,
			FeatureID:        feature.ID,
			FeatureKey:       feature.Key,
			UsageAttribution: usageAttribution,
			MeasureUsageFrom: convert.ToPointer(testutils.GetRFC3339Time(t, "1983-12-24T00:00:00Z")), // old, override in tests
			EntitlementType:  entitlement.EntitlementTypeMetered,
			IssueAfterReset:  convert.ToPointer(0.0),
			IsSoftLimit:      convert.ToPointer(false),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor:   getAnchor(t),
				Interval: timeutil.RecurrencePeriodYear,
			})),
		}

		currentUsagePeriod, err := input.UsagePeriod.GetValue().GetPeriodAt(time.Now()) // This should be calculated properly when testing batch resets
		require.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	connector, deps := setupConnector(t)
	defer deps.Teardown()

	// create featute in db
	feat, err := deps.featureRepo.CreateFeature(t.Context(), exampleFeature)
	require.NoError(t, err)
	require.NotEmpty(t, feat)

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should allow resetting usage for the first time with no grants",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)

				resetTime := startTime.Add(time.Hour * 3)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				startingBalance, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				require.NoError(t, err)

				assert.Equal(t, 0.0, startingBalance.UsageInPeriod) // cannot be usage
				assert.Equal(t, 0.0, startingBalance.Balance)       // no balance as there are no grants
				assert.Equal(t, 0.0, startingBalance.Overage)       // cannot be overage
			},
		},
		{
			name: "Should error if requested reset time is before start of measurement",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)
				assert.NotNil(t, ent)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// resetTime before start of measurement
				resetTime := startTime.Add(-time.Hour)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
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
				startTime := getAnchor(t)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				// save a reset time
				priorResetTime := startTime.Add(time.Hour)
				err = deps.usageResetRepo.Save(ctx, meteredentitlement.UsageResetUpdate{
					NamespacedModel:     models.NamespacedModel{Namespace: namespace},
					ResetTime:           priorResetTime,
					Anchor:              ent.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Anchor,
					EntitlementID:       ent.ID,
					UsagePeriodInterval: ent.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetValue().Interval.ISOString(),
				})
				require.NoError(t, err)

				// resetTime before prior reset time
				resetTime := priorResetTime.Add(-time.Minute)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				assert.ErrorContains(t, err, "is before current usage period start")
			},
		},
		{
			name: "Should error if requested reset time is in the future",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				now := time.Now().Truncate(time.Minute)
				aDayAgo := now.Add(-time.Hour * 24)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &aDayAgo
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

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
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// we force snapshot creation the intended way by checking the balance

				// issue grant
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// some usage on ledger, should be inconsequential
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				queryTime := startTime.Add(time.Hour * 5) // over grace period
				// we get the balance to force snapshot creation
				// We create a snapshot at the time of the grant
				clock.SetTime(g1.EffectiveAt)

				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}

				err = deps.balanceSnapshotService.Save(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, []balance.Snapshot{
					{
						At:      g1.EffectiveAt,
						Overage: 0,
						Balances: balance.Map{
							g1.ID: 1000,
						},
					},
				})
				require.NoError(t, err)
				clock.ResetTime()

				// for sanity check that snapshot was created (at g1.EffectiveAt)
				snap, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
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
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
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
				creditBalance, err := deps.balanceConnector.GetBalanceAt(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, resetTime)
				assert.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Snapshot.Balances)
			},
		},
		{
			name: "Should preserve overage after reset and deduct it from new balance",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    3,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
					// After each reset has a new 500 balance
					ResetMaxRollover: 500,
					ResetMinRollover: 500,
				})
				assert.NoError(t, err)

				// usage on ledger that will cause overage
				deps.streamingConnector.AddSimpleEvent(meterSlug, 2100, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:              resetTime,
						PreserveOverage: convert.ToPointer(true),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 400.0, balanceAfterReset.Balance)     // (1000 + 1000 - 2100) + 500 = 400
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // Overage is carried to new period
				assert.Equal(t, resetTime, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err := deps.balanceConnector.GetBalanceAt(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, resetTime)
				assert.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 0,
					g2.ID: 400,
				}, creditBalance.Snapshot.Balances)
			},
		},
		{
			name: "Should preserve overage after reset and deduct it from new balance resulting in overage at start of period",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				require.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    3,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
					// After each reset has a new 500 balance
					ResetMaxRollover: 500,
					ResetMinRollover: 500,
				})
				require.NoError(t, err)

				// usage on ledger that will cause overage
				deps.streamingConnector.AddSimpleEvent(meterSlug, 2600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:              resetTime,
						PreserveOverage: convert.ToPointer(true),
					})

				require.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 0.0, balanceAfterReset.Balance)       // (1000 + 1000 - 2600) + 500 = -100 => 0
				assert.Equal(t, 100.0, balanceAfterReset.Overage)     // Overage is carried to new period
				assert.Equal(t, resetTime, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err := deps.balanceConnector.GetBalanceAt(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, resetTime)
				require.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 0,
					g2.ID: 0,
				}, creditBalance.Snapshot.Balances)
			},
		},
		{
			name: "Should return proper last reset time after reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)
				assert.Equal(t, startTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// resetTime before snapshot
				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})
				require.NoError(t, err)

				// validate that lastReset time is properly set
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)
				assert.Equal(t, resetTime.Format(time.RFC3339), ent.LastReset.Format(time.RFC3339))
			},
		},
		{
			name: "Should calculate balance for grants taking effect after last saved snapshot",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// issue grants
				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				require.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         3,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 100, // full amount can be rolled over
				})
				require.NoError(t, err)

				// usage on ledger that will be deducted from g1
				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				// do a reset
				resetTime1 := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime1,
					})

				require.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 500.0, balanceAfterReset.Balance)     // 1000 - 600 = 400 rolled over + MAX(1000 - 0, 100)=100 = 500
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // no overage
				assert.Equal(t, resetTime1, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err := deps.balanceConnector.GetBalanceAt(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, resetTime1)
				require.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 400,
					g2.ID: 100,
				}, creditBalance.Snapshot.Balances)

				// issue grants taking effect after first reset
				g3, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      resetTime1.Add(time.Hour * 1),
					ExpiresAt:        lo.ToPtr(resetTime1.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				require.NoError(t, err)

				// add usage after reset 1
				deps.streamingConnector.AddSimpleEvent(meterSlug, 300, resetTime1.Add(time.Minute*10))

				// do a 2nd reset
				resetTime2 := resetTime1.Add(time.Hour * 5)
				balanceAfterReset, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime2,
					})

				require.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 1200.0, balanceAfterReset.Balance)    // 1000 + 500 - 300 = 1200
				assert.Equal(t, 0.0, balanceAfterReset.Overage)       // no overage
				assert.Equal(t, resetTime2, balanceAfterReset.StartOfPeriod)

				// get detailed balance from credit connector to check individual grant balances
				creditBalance, err = deps.balanceConnector.GetBalanceAt(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        ent.ID,
				}, resetTime2)
				require.NoError(t, err)

				assert.Equal(t, balance.Map{
					g1.ID: 100,
					g2.ID: 100,
					g3.ID: 1000,
				}, creditBalance.Snapshot.Balances)
			},
		},
		{
			name: "Should properly handle grants issued for the same time as reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)

				// add 0 usage so meter is found in mock
				deps.streamingConnector.AddSimpleEvent(meterSlug, 0, startTime)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())

				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 0, // full amount can be rolled over
				})
				require.NoError(t, err)

				// do a reset
				resetTime := startTime.Add(time.Hour * 5)
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				// assert balance after reset is 0 for grant
				require.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 0.0, balanceAfterReset.Balance)       // 1000 - 1000 = 0

				// issue grants
				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      resetTime,
					ExpiresAt:        lo.ToPtr(resetTime.AddDate(0, 0, 3)),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				require.NoError(t, err)

				// fetch balance for reset & grant, balance should be full grant amount
				balanceAfterReset, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, resetTime)
				require.NoError(t, err)

				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, g2.Amount, balanceAfterReset.Balance) // 1000 - 0 = 1000

				// fetch balance for AFTER reset & grant, balance should be full grant amount
				balanceAfterReset, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, resetTime.Add(time.Minute))
				require.NoError(t, err)

				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, g2.Amount, balanceAfterReset.Balance) // 1000 - 0 = 1000
			},
		},
		{
			name: "Should properly handle grants expiring the same time as reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := getAnchor(t)
				resetTime := startTime.AddDate(0, 0, 3)

				// add 0 usage so meter is found in mock
				deps.streamingConnector.AddSimpleEvent(meterSlug, 0, startTime)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          ent.ID,
					Namespace:        namespace,
					Amount:           1000,
					Priority:         1,
					EffectiveAt:      startTime.Add(time.Hour * 2),
					ExpiresAt:        lo.ToPtr(resetTime),
					ResetMaxRollover: 1000, // full amount can be rolled over
				})
				require.NoError(t, err)

				// do a reset
				balanceAfterReset, err := connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				// assert balance after reset is 0 for grant
				require.NoError(t, err)
				assert.Equal(t, 0.0, balanceAfterReset.UsageInPeriod) // 0 usage right after reset
				assert.Equal(t, 0.0, balanceAfterReset.Balance)       // Grant expires at reset time so we should see no balance
			},
		},
		{
			name: "Should reseting without anchor update keeps the next reset time intact",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Anchor:   anchor,
					Interval: timeutil.RecurrencePeriodDaily,
				}))
				inp.CurrentUsagePeriod = &timeutil.ClosedPeriod{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: true,
					})

				require.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)
				assertUsagePeriodInputsEquals(t, inp.UsagePeriod, ent.UsagePeriod.GetOriginalValueAsUsagePeriodInput())
			},
		},
		{
			name: "Should reseting with anchor update updates the next reset time too",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod = lo.ToPtr(entitlement.NewStartingUsagePeriodInput(timeutil.Recurrence{
					Anchor:   anchor,
					Interval: timeutil.RecurrencePeriodDaily,
				}, startTime))
				inp.CurrentUsagePeriod = &timeutil.ClosedPeriod{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				require.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)

				uInpNow, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(clock.Now())
				require.NoError(t, err)

				assertUsagePeriodInputsEquals(t, lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily,
					Anchor:   resetTime,
				})), lo.ToPtr(uInpNow))
			},
		},
		{
			name: "When resetting with anchor update the anchor gets truncated to per minute resolution",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := time.Now().Add(-12 * time.Hour).Truncate(time.Minute)
				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				anchor := startTime.Add(time.Hour)
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Anchor:   anchor,
					Interval: timeutil.RecurrencePeriodDaily,
				}))
				inp.CurrentUsagePeriod = &timeutil.ClosedPeriod{
					To: anchor.AddDate(0, 0, 1),
				}

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, startTime.Add(time.Minute))

				resetTime := startTime.Add(time.Hour * 5).Add(time.Second)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At: resetTime,
					})

				require.NoError(t, err)
				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)

				uInpNow, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(clock.Now())
				require.NoError(t, err)

				assertUsagePeriodInputsEquals(t, lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily,
					Anchor:   resetTime.Truncate(time.Minute),
				})), lo.ToPtr(uInpNow))
			},
		},
		{
			name: "Should be able to reset at a programmatic reset time",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				// Programmatic reset happens at midnight (DAILY recurrence)
				// Now its 12:01 am
				// And we reset for midnight
				ctx := context.Background()

				yesterdayMidnight := func() time.Time {
					t.Helper()
					now := clock.Now().UTC()
					return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
				}()

				startTime := yesterdayMidnight.Add(time.Minute)
				entitlementTime := yesterdayMidnight.AddDate(0, 0, -2)
				resetTime := yesterdayMidnight

				// Let's add usage so the meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, entitlementTime.Add(time.Minute))

				// Let's time-travel to the start time so resources have existed for a while
				clock.SetTime(entitlementTime)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &entitlementTime
				anchor := entitlementTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Anchor:   anchor,
					Interval: timeutil.RecurrencePeriodDaily,
				}))

				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// Let's time travel back to the current time
				clock.SetTime(startTime)

				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: false,
					})

				require.NoError(t, err)

				ent, err = deps.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID})
				require.NoError(t, err)

				uInpNow, _, err := ent.UsagePeriod.GetUsagePeriodInputAt(clock.Now())
				require.NoError(t, err)

				assertUsagePeriodInputsEquals(t, lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily,
					Anchor:   resetTime,
				})), lo.ToPtr(uInpNow))
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			deps.streamingConnector.Reset()
			tc.run(t, connector, deps)
		})
	}
}
