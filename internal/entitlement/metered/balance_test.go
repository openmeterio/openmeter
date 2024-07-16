package meteredentitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/testutils"
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

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriodAt(time.Now())
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should ignore usage before start of measurement",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				inp := getEntitlement(t, feature)
				inp.MeasureUsageFrom = &startTime
				// create entitlement in db
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(-time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, startTime.Add(time.Hour))
				assert.NoError(t, err)

				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 0.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage if there's no active grant",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				startTime := testutils.GetRFC3339Time(t, "2024-03-01T00:00:00Z")

				// create featute in db
				feature, err := deps.featureRepo.CreateFeature(ctx, exampleFeature)
				assert.NoError(t, err)

				// create entitlement in db
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, getEntitlement(t, feature))
				assert.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				assert.Equal(t, 100.0, entBalance.UsageInPeriod)
				assert.Equal(t, 100.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage until very first grant after reset",
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

				// add dummy usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 0, startTime.Add(-time.Minute))

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
				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, resetTime.Add(time.Minute))

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

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(entitlement.ID),
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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

				queryTime := startTime.Add(3 * time.Hour) // longer than grace period for saving snapshots

				// issue grants
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(entitlement.ID),
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    2,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour),
					ExpiresAt:   startTime.Add(time.Hour).AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, g1.EffectiveAt.Add(time.Minute*5))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, g2.EffectiveAt.Add(time.Minute))

				// add a balance snapshot
				err = deps.balanceSnapshotRepo.Save(
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
				snap1, err := deps.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, queryTime)
				assert.NoError(t, err)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 200.0, entBalance.UsageInPeriod) // in total we had 200 usage
				assert.Equal(t, 1550.0, entBalance.Balance)      // 750 + 1000 (g2 amount) - 200 = 1550
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, queryTime)
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

				queryTime := startTime.Add(3 * time.Hour) // longer than grace period for saving snapshots

				// issue grants
				owner := credit.NamespacedGrantOwner{
					Namespace: namespace,
					ID:        credit.GrantOwner(entitlement.ID),
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    2,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				g2, err := deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     owner.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour),
					ExpiresAt:   startTime.Add(time.Hour).AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, g1.EffectiveAt.Add(time.Minute*5))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, g2.EffectiveAt.Add(time.Minute))

				// add a balance snapshot
				err = deps.balanceSnapshotRepo.Save(
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
				snap1, err := deps.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, queryTime)
				assert.NoError(t, err)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				assert.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 200.0, entBalance.UsageInPeriod) // in total we had 200 usage
				assert.Equal(t, 1550.0, entBalance.Balance)      // 750 + 1000 (g2 amount) - 200 = 1550
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, queryTime)
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

				// FIXME: we shouldn't check things that the contract is unable to tell us
				snaps, err := deps.creditDBClient.BalanceSnapshot.Query().All(ctx)
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
			defer deps.Teardown()
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

		currentUsagePeriod, err := input.UsagePeriod.GetCurrentPeriodAt(time.Now())
		assert.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should return windowed history",
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

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				// issue grants
				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant falling on 3h window
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 3),
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// grant between windows
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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

				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
					OwnerID:     credit.GrantOwner(ent.ID),
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   startTime.AddDate(0, 0, 3),
				})
				assert.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

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
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				// grant after the reset
				_, err = deps.grantRepo.CreateGrant(ctx, credit.GrantRepoCreateGrantInput{
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
			defer deps.Teardown()
			tc.run(t, connector, deps)
		})
	}
}
