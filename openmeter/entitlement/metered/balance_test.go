package meteredentitlement_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	dbbalancesnapshot "github.com/openmeterio/openmeter/openmeter/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// We need to find a start time for our tests that is recent enough in our past
func getAnchor(t *testing.T) time.Time {
	t.Helper()
	now := clock.Now().UTC()
	return datetime.NewDateTime(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)).AddDateNoOverflow(0, -1, 0).Time
}

func TestGetEntitlementBalance(t *testing.T) {
	namespace := "ns1"
	meterSlug := "meter1"

	exampleFeature := feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature-1",
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
			MeasureUsageFrom: convert.ToPointer(testutils.GetRFC3339Time(t, "1983-03-01T00:00:00Z")), // old, override in tests
			EntitlementType:  entitlement.EntitlementTypeMetered,
			IssueAfterReset:  convert.ToPointer(0.0),
			IsSoftLimit:      convert.ToPointer(false),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor:   getAnchor(t),
				Interval: timeutil.RecurrencePeriodYear,
			})),
		}

		currentUsagePeriod, err := input.UsagePeriod.GetValue().GetPeriodAt(time.Now())
		require.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	connector, deps := setupConnector(t)
	defer deps.Teardown()

	// create featute in db
	feat, err := deps.featureRepo.CreateFeature(t.Context(), exampleFeature)
	require.NoError(t, err)

	_, err = deps.dbClient.Subject.Create().SetNamespace(namespace).SetKey("subject1").Save(t.Context())
	require.NoError(t, err)

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should ignore usage before start of measurement",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				// create entitlement in db
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(-time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, startTime.Add(time.Hour))
				require.NoError(t, err)

				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 0.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage if there's no active grant",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, getEntitlement(t, feat, cust.GetUsageAttribution()))
				require.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				assert.Equal(t, 100.0, entBalance.UsageInPeriod)
				assert.Equal(t, 100.0, entBalance.Overage)
			},
		},
		{
			name: "Should return overage until very first grant after reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()

				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

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
				require.NoError(t, err)

				// usage on ledger that will be deducted
				deps.streamingConnector.AddSimpleEvent(meterSlug, 600, resetTime.Add(time.Minute))

				// get balance with overage
				queryTime := resetTime.Add(time.Hour)
				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, queryTime)

				require.NoError(t, err)
				assert.Equal(t, 600.0, entBalance.UsageInPeriod)
				assert.Equal(t, 600.0, entBalance.Overage)
				assert.Equal(t, 0.0, entBalance.Balance)
			},
		},
		{
			name: "Should return correct usage and balance",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.Add(time.Hour)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(time.Minute))

				// issue grants
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     entitlement.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     entitlement.ID,
					Namespace:   namespace,
					Amount:      1000,
					Priority:    1,
					EffectiveAt: queryTime.Add(time.Hour),
					ExpiresAt:   lo.ToPtr(queryTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				assert.Equal(t, 100.0, entBalance.UsageInPeriod)
				assert.Equal(t, 900.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)
			},
		},
		{
			name: "Should save new snapshot when querying period start",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				// register usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, startTime.AddDate(5, 0, 0))

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily, // We use a faster recurrence so it overwrites the grace periods
					Anchor:   inp.UsagePeriod.GetValue().Anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 0, 9) // falls on period start

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         2,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 10)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute))

				// add a balance snapshot
				err = deps.balanceSnapshotService.Save(
					ctx,
					owner, []balance.Snapshot{
						{
							Usage: balance.SnapshottedUsage{
								Since: startTime,
								Usage: 0,
							},
							Balances: balance.Map{
								g1.ID: 1000,
							},
							Overage: 0,
							At:      g1.EffectiveAt,
						},
					})
				require.NoError(t, err)

				clock.SetTime(queryTime)

				// get last vaild snapshot
				snap1, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: startTime,
						Usage: 0,
					},
					Balances: balance.Map{
						g1.ID: 1000,
					},
					Overage: 0,
					At:      g1.EffectiveAt,
				}, snap1)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				// check snapshots
				assert.NotEqual(t, snap1.At, snap2.At)
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: queryTime, // querytime is the start of a UsagePeriod, so this snapshot will be at the start of the usage period
						Usage: 0,         // And at a reset time the usage is 0
					},
					Balances: balance.Map{
						g1.ID: 800,
					},
					Overage: 0,
					At:      queryTime, // querytime is the start of a UsagePeriod, so we create a snapshot
				}, snap2)
			},
		},
		{
			name: "Should save new snapshot at start of current period if period start is closer than graceperiod",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				// register usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, startTime.AddDate(5, 0, 0))

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily, // We use a faster recurrence so it overwrites the grace periods
					Anchor:   inp.UsagePeriod.GetValue().Anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 0, 9).Add(time.Hour) // queried an after hour period start (which is daily)

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         2,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 10)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute))

				// add a balance snapshot
				err = deps.balanceSnapshotService.Save(
					ctx,
					owner, []balance.Snapshot{
						{
							Usage: balance.SnapshottedUsage{
								Since: startTime,
								Usage: 0,
							},
							Balances: balance.Map{
								g1.ID: 1000,
							},
							Overage: 0,
							At:      g1.EffectiveAt,
						},
					})
				require.NoError(t, err)

				clock.SetTime(queryTime)

				// get last vaild snapshot
				snap1, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: startTime,
						Usage: 0,
					},
					Balances: balance.Map{
						g1.ID: 1000,
					},
					Overage: 0,
					At:      g1.EffectiveAt,
				}, snap1)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				// check snapshots
				assert.NotEqual(t, snap1.At, snap2.At)
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: startTime.AddDate(0, 0, 9), // will create a snapshot at the start of the usage period
						Usage: 0,                          // And at a reset time the usage is 0
					},
					Balances: balance.Map{
						g1.ID: 800,
					},
					Overage: 0,
					At:      startTime.AddDate(0, 0, 9), // will create a snapshot at the start of the usage period
				}, snap2)
			},
		},
		{
			name: "Should save new snapshot at last (non-deterministic) history breakpoint before the grace period (even if there are no previous snapshots available)",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				// register usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, startTime.AddDate(5, 0, 0))

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				// Anchor is now() - 1 months - 1 months
				// But without overflow, example:
				//   now = 2026-01-31
				//   startTime = 2025-12-31
				//   anchor = 2025-11-30

				anchor := inp.UsagePeriod.GetValue().Anchor
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth, // We use a slower recurrence so grace period will fit inside it
					Anchor:   anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := datetime.NewDateTime(startTime).AddDateNoOverflow(0, 1, 9).Time // We progress into the next month when querying so we'll have a history breakpoint at reset time

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         2,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(datetime.NewDateTime(startTime).AddDateNoOverflow(0, 1, 10).Time), // let's also extend this a month
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute))

				// Let's not save a snapshot to simulate a clean-slate scenario

				clock.SetTime(queryTime)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				snap, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				// check snapshot
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: datetime.NewDateTime(anchor).AddDateNoOverflow(0, 2, 0).Time, // Will create a snapshot at the last history breakpoint outside 7 day grace period (which is the last reset time)
						Usage: 0,                                                            // And at a reset time the usage is 0
					},
					Balances: balance.Map{
						g1.ID: 800,
					},
					Overage: 0,
					At:      datetime.NewDateTime(anchor).AddDateNoOverflow(0, 2, 0).Time, // Will create a snapshot at the last history breakpoint outside 7 day grace period (which is the last reset time)
				}, snap)
			},
		},
		{
			// This behavious is not ideal but would require a longer refactor to change. In a scenario where we're in the first usage period and there either haven't been any grants issued, or they were issued same time as start of measurement, we will only have a single history segment in the result which will result in us not snapshotting.
			// This is not a significant issue as with the next periods we'll have a history breakpoint and start saving snapshots again. This is asserted in the test above [Should save new snapshot at last (non-deterministic) history breakpoint before the grace period (even if there are no previous snapshots available)]
			name: "Will not save a snapshot if there are no previous snapshots and there have been no significant events since start of measurement (i.e. reset / grant issuing)",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				// register usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, startTime.AddDate(5, 0, 0))

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   inp.UsagePeriod.GetValue().Anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 0, 9) // we'll only have a breakpoint at the usage period boundaries which is starttime, 9 days is longer than grace period (7 days)

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         2,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 0, 10)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute))

				// don't add a balance snapshot (we'll calculate from start of time)

				clock.SetTime(queryTime)

				// get last vaild snapshot, assert there isn't one
				_, err = deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.ErrorAs(t, err, lo.ToPtr(&balance.NoSavedBalanceForOwnerError{}))

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 200.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				// will not save a snapshot as there were no history breakpoints
				_, err = deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.ErrorAs(t, err, lo.ToPtr(&balance.NoSavedBalanceForOwnerError{}))
			},
		},
		{
			name: "Should save snapshot with correct usage data for period",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				// TODO: let's revert this once we have fixed the period calculation
				// startTime := getAnchor(t)
				startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				// register usage so meter is found
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1, startTime.AddDate(5, 0, 0)) // far in future

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   inp.UsagePeriod.GetValue().Anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 1, 9) // will fall in next usageperiod

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         1,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(startTime.AddDate(1, 0, 0)), // far future
				})
				require.NoError(t, err)

				// add a balance snapshot
				err = deps.balanceSnapshotService.Save(
					ctx,
					owner, []balance.Snapshot{
						{
							Usage: balance.SnapshottedUsage{
								Since: startTime,
								Usage: 0,
							},
							Balances: balance.Map{
								g1.ID: 1000,
							},
							Overage: 0,
							At:      g1.EffectiveAt,
						},
					})
				require.NoError(t, err)

				// register usage for meter & feature in first period
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute))

				// register usage for meter & feature in second period
				clock.SetTime(startTime.AddDate(0, 1, 1))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, startTime.AddDate(0, 1, 1))

				// We need another event so there's a history breakpoint. Let's create another grant
				g2, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         10,
					EffectiveAt:      startTime.AddDate(0, 1, 2), // After the second round of usage is in
					ExpiresAt:        lo.ToPtr(startTime.AddDate(1, 0, 0)),
				})
				require.NoError(t, err)

				// register usage for meter & feature in second period after grant
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, startTime.AddDate(0, 1, 3))

				clock.SetTime(queryTime)

				// get last vaild snapshot
				snap1, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)
				// Should be the first and only snapshot we created
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: startTime,
						Usage: 0,
					},
					Balances: balance.Map{
						g1.ID: 1000,
					},
					Overage: 0,
					At:      g1.EffectiveAt,
				}, snap1)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 400.0, entBalance.UsageInPeriod)
				assert.Equal(t, 1400.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				// check snapshots
				assert.NotEqual(t, snap1.At, snap2.At)
				assert.Equal(t, balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: startTime.AddDate(0, 1, 0), // The programmatic reset time
						Usage: 200,                        // Total usage in second period so far
					},
					Balances: balance.Map{
						g1.ID: 600,
						g2.ID: 1000,
					},
					Overage: 0,
					At:      g2.EffectiveAt, // Our period start time
				}, snap2)
			},
		},
		{
			name: "Should not save the same snapshot over and over again",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)
				clock.SetTime(startTime)
				defer clock.ResetTime()

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				inp.UsagePeriod = lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily, // we need a faster recurrence as we wont save snapshots in the current usage period
					Anchor:   inp.UsagePeriod.GetValue().Anchor,
				}))
				entitlement, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 0, 10) // longer than grace period for saving snapshots

				// issue grants
				owner := models.NamespacedID{
					Namespace: namespace,
					ID:        entitlement.ID,
				}

				g1, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:          entitlement.ID,
					Namespace:        namespace,
					Amount:           1000,
					ResetMaxRollover: 1000,
					Priority:         2,
					EffectiveAt:      startTime,
					ExpiresAt:        lo.ToPtr(startTime.AddDate(0, 5, 0)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 200, g1.EffectiveAt.Add(time.Minute*5))

				// add a balance snapshot
				err = deps.balanceSnapshotService.Save(
					ctx,
					owner, []balance.Snapshot{
						{
							Balances: balance.Map{
								g1.ID: 1000,
							},
							Overage: 0,
							At:      g1.EffectiveAt,
						},
					})
				require.NoError(t, err)

				// get last vaild snapshot
				snap1, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				clock.SetTime(queryTime)

				entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				snap2, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryTime)
				require.NoError(t, err)

				// check snapshots
				assert.NotEqual(t, snap1.At, snap2.At)
				assert.Equal(t, 0.0, snap2.Overage)
				assert.Equal(t, balance.Map{
					g1.ID: 800,
				}, snap2.Balances)

				// run the calc again
				entBalance, err = connector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: namespace, ID: entitlement.ID}, queryTime)
				require.NoError(t, err)

				// validate balance calc for good measure
				assert.Equal(t, 0.0, entBalance.UsageInPeriod)
				assert.Equal(t, 800.0, entBalance.Balance)
				assert.Equal(t, 0.0, entBalance.Overage)

				// FIXME: we shouldn't check things that the contract is unable to tell us
				snaps, err := deps.dbClient.BalanceSnapshot.
					Query().
					Where(dbbalancesnapshot.OwnerID(entitlement.ID)).
					All(ctx)
				require.NoError(t, err)
				assert.Len(t, snaps, 2) // one for the initial and one we made last time
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

func TestGetEntitlementHistory(t *testing.T) {
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
			MeasureUsageFrom: convert.ToPointer(testutils.GetRFC3339Time(t, "1983-01-05T00:00:00Z")), // old, override in tests
			EntitlementType:  entitlement.EntitlementTypeMetered,
			IssueAfterReset:  convert.ToPointer(0.0),
			IsSoftLimit:      convert.ToPointer(false),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor:   getAnchor(t),
				Interval: timeutil.RecurrencePeriodYear,
			})),
		}

		currentUsagePeriod, err := input.UsagePeriod.GetValue().GetPeriodAt(time.Now())
		require.NoError(t, err)
		input.CurrentUsagePeriod = &currentUsagePeriod
		return input
	}

	connector, deps := setupConnector(t)
	defer deps.Teardown()

	_, _ = deps.dbClient.Subject.Create().SetNamespace(namespace).SetKey("subject1").Save(t.Context())

	// create featute in db
	feat, err := deps.featureRepo.CreateFeature(t.Context(), exampleFeature)
	require.NoError(t, err)

	tt := []struct {
		name string
		run  func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should return windowed history",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

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
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// grant falling on 3h window
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 3),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// grant between windows
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				t.Run("Should return correct value for the entire period", func(t *testing.T) {
					windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
						From:           &startTime,
						To:             &queryTime,
						WindowTimeZone: *time.UTC,
						WindowSize:     meteredentitlement.WindowSizeHour,
					})
					require.NoError(t, err)

					assert.Len(t, windowedHistory, 12)

					// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
					assert.Equal(t, 100.0, windowedHistory[0].UsageInPeriod)
					assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
					assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
					assert.Zero(t, startTime.Add(time.Hour).Compare(windowedHistory[1].From))
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
				})

				t.Run("Should truncate input period to meter window size", func(t *testing.T) {
					windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
						From:           lo.ToPtr(startTime.Add(2 * time.Second)),
						To:             lo.ToPtr(queryTime.Add(-2 * time.Second)),
						WindowTimeZone: *time.UTC,
						WindowSize:     meteredentitlement.WindowSizeHour,
					})
					require.NoError(t, err)

					assert.Len(t, windowedHistory, 12)

					// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
					assert.Equal(t, 100.0, windowedHistory[0].UsageInPeriod)
					assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
					assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
					assert.Zero(t, startTime.Add(time.Hour).Compare(windowedHistory[1].From))
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
				})

				t.Run("Should return correct value if the queried period doesn't coincide with history breakpoints", func(t *testing.T) {
					windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
						From:           lo.ToPtr(startTime.Add(time.Hour)),
						To:             &queryTime,
						WindowTimeZone: *time.UTC,
						WindowSize:     meteredentitlement.WindowSizeHour,
					})
					require.NoError(t, err)

					// check returned burndownhistory
					segments := burndownHistory.Segments()
					assert.Len(t, segments, 3)

					assert.Zero(t, segments[0].From.Compare(startTime.Add(time.Hour)))
					assert.Equal(t, 9900.0, segments[0].BalanceAtStart.Balance())

					// check windowed history
					assert.Len(t, windowedHistory, 11)

					assert.Zero(t, startTime.Add(time.Hour).Compare(windowedHistory[0].From))
					assert.Equal(t, 9900.0, windowedHistory[0].BalanceAtStart)
					// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
					assert.Equal(t, 100.0, windowedHistory[1].UsageInPeriod)
					assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
					// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
					assert.Equal(t, 100.0, windowedHistory[2].UsageInPeriod)
					assert.Equal(t, 19800.0, windowedHistory[2].BalanceAtStart)
					assert.Equal(t, 19700.0, windowedHistory[3].BalanceAtStart)
					// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
					assert.Equal(t, 100.0, windowedHistory[4].UsageInPeriod)
					assert.Equal(t, 19700.0, windowedHistory[4].BalanceAtStart) // even though EffectiveAt: startTime.Add(time.Hour * 5).Add(time.Minute * 30) grant happens here, it is only recognized at the next window
					assert.Equal(t, 29600.0, windowedHistory[5].BalanceAtStart)
					assert.Equal(t, 29600.0, windowedHistory[6].BalanceAtStart)
					// deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
					assert.Equal(t, 1100.0, windowedHistory[7].UsageInPeriod)
					assert.Equal(t, 29600.0, windowedHistory[7].BalanceAtStart)
					assert.Equal(t, 28500.0, windowedHistory[8].BalanceAtStart)
					// deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))
					assert.Equal(t, 100.0, windowedHistory[10].UsageInPeriod)
					assert.Equal(t, 28500.0, windowedHistory[10].BalanceAtStart)
				})
			},
		},
		{
			name: "If start time is not specified we are defaulting to the last reset",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

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
				require.NoError(t, err)

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				// grant after the reset
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: resetTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					To:             &queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeHour,
				})
				require.NoError(t, err)

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
				assert.Len(t, segments, 1)
			},
		},
		{
			name: "If start time is not specified we are defaulting to NEXT WINDOW after start of measurement if there were no manual resets",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// grant again later
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					To:             &queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeHour,
				})
				require.NoError(t, err)

				assert.Len(t, windowedHistory, 12)

				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[0].UsageInPeriod)
				assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
				assert.Equal(t, 0.0, windowedHistory[1].UsageInPeriod)
				assert.Equal(t, 9900.0, windowedHistory[1].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[2].UsageInPeriod)
				assert.Equal(t, 19900.0, windowedHistory[2].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[3].UsageInPeriod)
				assert.Equal(t, 19800.0, windowedHistory[3].BalanceAtStart)
				assert.Equal(t, 19700.0, windowedHistory[4].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[5].UsageInPeriod)
				assert.Equal(t, 19700.0, windowedHistory[5].BalanceAtStart)
				assert.Equal(t, 19600.0, windowedHistory[6].BalanceAtStart)
				assert.Equal(t, 19600.0, windowedHistory[7].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				assert.Equal(t, 1100.0, windowedHistory[8].UsageInPeriod)
				assert.Equal(t, 19600.0, windowedHistory[8].BalanceAtStart)
				assert.Equal(t, 18500.0, windowedHistory[9].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))
				assert.Equal(t, 100.0, windowedHistory[11].UsageInPeriod)
				assert.Equal(t, 18500.0, windowedHistory[11].BalanceAtStart)

				// check returned burndownhistory
				segments := burndownHistory.Segments()
				assert.Len(t, segments, 2)
			},
		},
		{
			name: "If start time is not specified we are defaulting to NEXT WINDOWED after start of measurement if there were no manual resets and measurement starts not at a window boundary",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				startOfMeasurement := startTime.Add(time.Minute * 29)
				inp.MeasureUsageFrom = lo.ToPtr(startOfMeasurement)
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startOfMeasurement,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// grant again later
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime.Add(time.Hour * 2),
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startOfMeasurement.Add(time.Minute))

				queryTime := startTime.Add(time.Hour * 12)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					To:             &queryTime,
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeHour,
				})
				require.NoError(t, err)

				assert.Len(t, windowedHistory, 11)

				// deps.streaming.AddSimpleEvent(meterSlug, 100, startOfMeasurement.Add(time.Minute))
				assert.Equal(t, startTime.Add(time.Hour), windowedHistory[0].From.UTC())
				assert.Equal(t, 0.0, windowedHistory[0].UsageInPeriod)
				assert.Equal(t, 9900.0, windowedHistory[0].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*2).Add(time.Minute))
				assert.Equal(t, startTime.Add(time.Hour*2), windowedHistory[1].From.UTC())
				assert.Equal(t, 100.0, windowedHistory[1].UsageInPeriod)
				assert.Equal(t, 19900.0, windowedHistory[1].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*3).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[2].UsageInPeriod)
				assert.Equal(t, 19800.0, windowedHistory[2].BalanceAtStart)
				assert.Equal(t, 19700.0, windowedHistory[3].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour*5).Add(time.Minute))
				assert.Equal(t, 100.0, windowedHistory[4].UsageInPeriod)
				assert.Equal(t, 19700.0, windowedHistory[4].BalanceAtStart)
				assert.Equal(t, 19600.0, windowedHistory[5].BalanceAtStart)
				assert.Equal(t, 19600.0, windowedHistory[6].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*8).Add(time.Minute))
				assert.Equal(t, 1100.0, windowedHistory[7].UsageInPeriod)
				assert.Equal(t, 19600.0, windowedHistory[7].BalanceAtStart)
				assert.Equal(t, 18500.0, windowedHistory[8].BalanceAtStart)
				// deps.streaming.AddSimpleEvent(meterSlug, 100, queryTime.Add(-time.Second))
				assert.Equal(t, 100.0, windowedHistory[10].UsageInPeriod)
				assert.Equal(t, 18500.0, windowedHistory[10].BalanceAtStart)

				// check returned burndownhistory
				segments := burndownHistory.Segments()
				assert.Len(t, segments, 2)
			},
		},
		{
			name: "Should return history if WINDOWSIZE and entitlements events dont align",
			run: func(t *testing.T, connector meteredentitlement.Connector, deps *dependencies) {
				ctx := t.Context()
				startTime := getAnchor(t)

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// create entitlement in db
				inp := getEntitlement(t, feat, cust.GetUsageAttribution())
				inp.MeasureUsageFrom = &startTime
				ent, err := deps.entitlementRepo.CreateEntitlement(ctx, inp)
				require.NoError(t, err)

				// We'll query with WINDOWSIZE_DAY, so lets use 12h precision for the different events

				// grant at start
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      10000,
					Priority:    1,
					EffectiveAt: startTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 100, startTime.Add(time.Hour))

				// let's do a reset
				resetTime := startTime.Add(time.Hour * 12)
				_, err = connector.ResetEntitlementUsage(ctx,
					models.NamespacedID{Namespace: namespace, ID: ent.ID},
					meteredentitlement.ResetEntitlementUsageParams{
						At:           resetTime,
						RetainAnchor: true,
					},
				)
				require.NoError(t, err)

				queryTime := startTime.AddDate(0, 0, 2)

				// register usage for meter & feature
				deps.streamingConnector.AddSimpleEvent(meterSlug, 500, startTime.Add(time.Hour*11))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 300, startTime.Add(time.Hour*18))
				deps.streamingConnector.AddSimpleEvent(meterSlug, 1100, startTime.Add(time.Hour*25))

				// grant after the reset
				_, err = deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
					OwnerID:     ent.ID,
					Namespace:   namespace,
					Amount:      7000,
					Priority:    1,
					EffectiveAt: resetTime,
					ExpiresAt:   lo.ToPtr(startTime.AddDate(0, 0, 3)),
				})
				require.NoError(t, err)

				windowedHistory, burndownHistory, err := connector.GetEntitlementBalanceHistory(ctx, models.NamespacedID{Namespace: namespace, ID: ent.ID}, meteredentitlement.BalanceHistoryParams{
					To:             &queryTime,
					From:           lo.ToPtr(startTime),
					WindowTimeZone: *time.UTC,
					WindowSize:     meteredentitlement.WindowSizeDay,
				})
				require.NoError(t, err)

				// check returned burndownhistory
				segments := burndownHistory.Segments()
				assert.Len(t, segments, 2)

				assert.Len(t, windowedHistory, 2)

				// First Day
				assert.Equal(t, 900.0, windowedHistory[0].UsageInPeriod)
				assert.Equal(t, 10000.0, windowedHistory[0].BalanceAtStart)
				// Second Day
				assert.Equal(t, 1100.0, windowedHistory[1].UsageInPeriod)
				assert.Equal(t, 6700.0, windowedHistory[1].BalanceAtStart)
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
