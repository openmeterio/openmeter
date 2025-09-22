package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestEngine(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)
	meterSlug := "meter-1"

	defaultMeter := meterpkg.Meter{
		Key:         meterSlug,
		Aggregation: meterpkg.MeterAggregationSum,
	}

	grant1 := makeGrant(grant.Grant{
		ID:          "grant-1",
		Amount:      100.0,
		Priority:    1,
		EffectiveAt: t1,
		Expiration: &grant.ExpirationPeriod{
			Duration: grant.ExpirationPeriodDurationDay,
			Count:    30,
		},
	})

	grant2 := makeGrant(grant.Grant{
		ID:          "grant-2",
		Amount:      100.0,
		Priority:    1,
		EffectiveAt: t1,
		Expiration: &grant.ExpirationPeriod{
			Duration: grant.ExpirationPeriodDurationDay,
			Count:    30,
		},
	})

	// Tests with single engine
	tt := []struct {
		name  string
		meter meterpkg.Meter
		run   func(t *testing.T, engine engine.Engine, use addUsageFunc, mm meterpkg.Meter)
	}{
		{
			name: "Should return the same result on subsequent runs",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1 = makeGrant(g1)

				res1, err1 := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1).Add(time.Hour),
					})
				assert.NoError(t, err1)

				res2, err2 := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1).Add(time.Hour),
					})
				assert.NoError(t, err2)

				assert.Equal(t, res1, res2)
			},
		},
		{
			name: "Reports overage if there are no grants",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{},
							Overage:  0,
							At:       t1,
						},
						Until: t1.AddDate(0, 0, 30),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res.Snapshot.Overage)
				assert.Equal(t, balance.Map{}, res.Snapshot.Balances)
				assert.Equal(t, []engine.GrantBurnDownHistorySegment{
					{
						BalanceAtStart: balance.Map{},
						GrantUsages:    []engine.GrantUsage{},
						ClosedPeriod: timeutil.ClosedPeriod{
							From: t1,
							To:   t1.AddDate(0, 0, 30),
						},
						TerminationReasons: engine.SegmentTerminationReason{},
						TotalUsage:         50.0,
						Overage:            50.0,
					},
				}, res.History.Segments())
			},
		},
		{
			name: "Errors if balance was provided for nonexistent grants",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				_, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								grant1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 30),
					})

				assert.Error(t, err)
			},
		},
		{
			name: "Errors on missing balance for one of the grants",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				g1 := grant1
				g1 = makeGrant(g1)
				g2 := grant2
				g2 = makeGrant(g2)
				_, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								grant1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 30),
					})

				assert.Error(t, err)
			},
		},
		{
			name: "Should do nothing for 0 length period",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// We have report usage so the meter is found
				use(100.0, t1.Add(time.Hour))

				prevPeriodStart := t1.AddDate(0, 0, -1)

				u := balance.SnapshottedUsage{
					Since: prevPeriodStart,
					Usage: 10.0,
				}

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{grant1},
						StartingSnapshot: balance.Snapshot{
							Usage: u,
							Balances: balance.Map{
								grant1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1,
					},
				)
				assert.NoError(t, err)
				assert.Equal(t, balance.Snapshot{
					Usage: u, // Should pass through the original usage info
					Balances: balance.Map{
						grant1.ID: 100.0,
					},
					Overage: 0,
					At:      t1,
				}, res.Snapshot)
			},
		},
		{
			name: "Able to burn down single active grant",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{grant1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								grant1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 30),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Return 0 balance for grant with future effectiveAt",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				g := grant1
				g.EffectiveAt = t1.AddDate(0, 0, 10)
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Return 0 balance for deleted grant",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				g := grant1
				g.EffectiveAt = t1
				g.DeletedAt = &t1
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Burns down grant until it's deleted",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				deletionTime := t1.Add(time.Hour)
				use(50.0, deletionTime.Add(-time.Minute))
				use(50.0, deletionTime.Add(time.Minute))
				g := grant1
				g.EffectiveAt = t1
				g.DeletedAt = &deletionTime
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 50.0, res.Snapshot.Overage) // usage after grant deletion
				assert.Len(t, res.History.Segments(), 2)
				assert.Equal(t, 50.0, res.History.Segments()[0].TotalUsage)
				assert.Equal(t, 50.0, res.History.Segments()[1].TotalUsage)
				assert.Equal(t, 0.0, res.History.Segments()[1].BalanceAtStart[g.ID])
			},
		},
		{
			name: "Burns down grant until it's voided",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				voidingTime := t1.Add(time.Hour)
				use(50.0, voidingTime.Add(-time.Minute))
				use(50.0, voidingTime.Add(time.Minute))
				g := grant1
				g.EffectiveAt = t1
				g.VoidedAt = &voidingTime
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 50.0, res.Snapshot.Overage) // usage after grant deletion
				assert.Len(t, res.History.Segments(), 2)
				assert.Equal(t, 50.0, res.History.Segments()[0].TotalUsage)
				assert.Equal(t, 50.0, res.History.Segments()[1].TotalUsage)
				assert.Equal(t, 0.0, res.History.Segments()[1].BalanceAtStart[g.ID])
			},
		},
		{
			name: "Return 0 balance for grant with past expiresAt",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				g := grant1
				g.EffectiveAt = t1.AddDate(-1, 0, 0)
				g.Expiration.Duration = grant.ExpirationPeriodDurationDay
				g.Expiration.Count = 1
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Does not burn down grant that expires at start of period",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				use(50.0, t1.Add(time.Hour))
				g := grant1
				g.EffectiveAt = t1.AddDate(-1, 0, 0)
				g.Expiration.Duration = grant.ExpirationPeriodDurationDay
				g.Expiration.Count = 1
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							Overage: 0,
							At:      *g.ExpiresAt,
						},
						Until: g.ExpiresAt.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Burns down grant that becomes active during phase",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				use(25.0, t1.Add(time.Hour))
				// burn down with usage prior to grant effectiveAt
				use(25.0, t1.Add(-time.Hour))
				g := grant1
				g.EffectiveAt = t1
				g.Expiration.Duration = grant.ExpirationPeriodDurationDay
				g.Expiration.Count = 30
				g = makeGrant(g)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 0.0,
							},
							Overage: 0,
							At:      t1.AddDate(0, 0, -1),
						},
						Until: t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res.Snapshot.Balances[grant1.ID])

				// sets correct starting balance for grant in segment
				assert.Len(t, res.History.Segments(), 2)
				assert.Equal(t, 0.0, res.History.Segments()[0].BalanceAtStart[g.ID])

				// starting balance doesnt have overage deducted
				assert.Equal(t, g.Amount, res.History.Segments()[1].BalanceAtStart[g.ID])
				// but it is stored separately
				assert.Equal(t, 25.0, res.History.Segments()[1].OverageAtStart)
			},
		},
		{
			name: "Burns down multiple grants",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				use(200, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t1
				g2 = makeGrant(g2)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant2.ID])
			},
		},
		{
			name: "Burns down grant with higher priority first",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1.Priority = 1
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t1
				g2.Priority = 2
				g2 = makeGrant(g2)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g2, g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 80.0, res.Snapshot.Balances[grant2.ID])
			},
		},
		{
			name: "Burns down grant that expires first",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t1
				g2.Expiration.Duration = grant.ExpirationPeriodDurationYear
				g2.Expiration.Count = 100
				g2 = makeGrant(g2)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g2, g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 80.0, res.Snapshot.Balances[grant2.ID])
			},
		},
		{
			name: "Burns down grant that expires first among many",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// make lots of grants
				nGrant := func(i int) grant.Grant {
					return makeGrant(grant.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      100.0,
						Priority:    1,
						EffectiveAt: t1,
						Expiration: &grant.ExpirationPeriod{
							Duration: grant.ExpirationPeriodDurationDay,
							Count:    30,
						},
					})
				}

				numGrants := 100000
				grants := make([]grant.Grant, 0, numGrants)
				for i := 0; i < numGrants; i++ {
					grants = append(grants, nGrant(i))
				}

				// set exp soonner on first grant
				grants[0].Expiration.Count = 29
				grants[0] = makeGrant(grants[0])

				gToBurn := grants[0]
				bm := balance.Map{}
				for _, g := range grants {
					bm[g.ID] = 100.0
				}

				// shuffle grants
				rand.Shuffle(len(grants), func(i, j int) {
					grants[i], grants[j] = grants[j], grants[i]
				})

				// burn down with usage after grant effectiveAt
				use(99, t1.Add(time.Hour))

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: grants,
						StartingSnapshot: balance.Snapshot{
							Balances: bm,
							Overage:  0,
							At:       t1,
						},
						Until: t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				for _, g := range grants {
					if g.ID == gToBurn.ID {
						assert.Equal(t, 1.0, res.Snapshot.Balances[g.ID])
					} else {
						assert.Equal(t, 100.0, res.Snapshot.Balances[g.ID])
					}
				}
			},
		},
		{
			name: "Burns down recurring grant",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1.Recurrence = &timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily,
					Anchor:   t1,
				}
				g1 = makeGrant(g1)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1).Add(time.Hour),
					})

				assert.NoError(t, err)
				// grant recurrs daily, so it should be 80.0
				assert.Equal(t, 80.0, res.Snapshot.Balances[grant1.ID])
			},
		},
		{
			name: "Burns down recurring grant that takes effect later before it has recurred",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				start := t1
				tg1 := start.AddDate(0, 0, -1)
				tg2 := start.AddDate(0, 0, 1)
				tg2r := start.AddDate(0, 0, 3)

				use(20, start.Add(-time.Hour))                    // g1
				use(20, start.Add(time.Hour))                     // g1
				use(20, start.AddDate(0, 0, 1).Add(-time.Second)) // g1 as its last period
				use(20, start.AddDate(0, 0, 1))                   // g2 due to priority and already effective (only matters before tg2r)

				g1 := grant1
				g1.EffectiveAt = tg1
				g1.Priority = 3
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				g2.Recurrence = &timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				res1, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 80.0, // due to use before start
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 10), // right after recurrence
					})

				assert.NoError(t, err)

				assert.Equal(t, 40.0, res1.Snapshot.Balances[grant1.ID])
				assert.Equal(t, 100.0, res1.Snapshot.Balances[grant2.ID]) // recurred just now (t1 + day10)
			},
		},
		{
			name: "Burns down recurring grant that takes effect later after it has recurred",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				start := t1
				tg1 := start.AddDate(0, 0, -1)
				tg2 := start.AddDate(0, 0, 1)
				tg2r := start.AddDate(0, 0, 3)

				use(20, start.Add(-time.Hour))                    // g1
				use(20, start.Add(time.Hour))                     // g1
				use(20, start.AddDate(0, 0, 1).Add(-time.Second)) // g1 as its last period
				use(20, tg2)                                      // g2 due to priority and already effective (only matters before tg2r)
				use(20, tg2r)                                     // g2 after first recurrence

				g1 := grant1
				g1.EffectiveAt = tg1
				g1.Priority = 3
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				g2.Recurrence = &timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				res2, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 80.0, // due to use before start
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 10).Add(-time.Hour), // right before recurrence
					})

				assert.NoError(t, err)

				assert.Equal(t, 40.0, res2.Snapshot.Balances[grant1.ID])
				// 20 usage as above
				assert.Equal(t, 80.0, res2.Snapshot.Balances[grant2.ID])
			},
		},
		{
			name: "Return GrantBurnDownHistory",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				start := t1
				end := start.AddDate(0, 1, 0)
				tg1 := start.AddDate(0, 0, 5)
				tg2 := start.AddDate(0, 0, 20)
				tg2r := start.AddDate(0, 0, 25) // recurrs on 25 then 32 which is past end time

				use(50, start.Add(time.Hour)) // usage between start and g1
				use(30, tg1.Add(time.Hour))   // usage between g1 and g2
				use(30, tg2.Add(time.Hour))   // usage between g2 and g2 recurrs
				use(20, tg2r.Add(time.Hour))  // usage after g2 recurrs

				g1 := grant1
				g1.EffectiveAt = tg1
				g1.Priority = 3
				// so they dont expire
				g1.Expiration.Count = 2
				g1.Expiration.Duration = grant.ExpirationPeriodDurationMonth
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				// so they dont expire
				g2.Expiration.Count = 2
				g2.Expiration.Duration = grant.ExpirationPeriodDurationMonth
				g2.Recurrence = &timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 80.0, // due to use before start
								g2.ID: 100.0,
							},
							Overage: 0,
							At:      start,
						},
						Until: end,
					})

				assert.NoError(t, err)

				assert.NotEmpty(t, res.History)
				assert.Equal(t, 4, len(res.History.Segments()))
			},
		},
		{
			name: "Should calculate sequential periods across timezones and convert to UTC",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				start := t1
				t1 := start.Add(time.Hour)
				t2 := t1.Add(time.Minute)
				t3 := t2.Add(8 * time.Minute)
				end := t3.Add(time.Hour)

				// set TZs
				loc1, err := time.LoadLocation("America/New_York")
				require.NoError(t, err)
				loc2, err := time.LoadLocation("Europe/Budapest")
				require.NoError(t, err)
				loc3, err := time.LoadLocation("Asia/Tokyo")
				require.NoError(t, err)

				t1 = t1.In(loc1)
				t2 = t2.In(loc2)
				t3 = t3.In(loc3)

				use(0, start.Add(time.Minute)) // we need usage so everything's found

				g1 := grant1
				g1.EffectiveAt = t1
				g1.Priority = 3
				g1.Expiration.Count = 1
				g1.Expiration.Duration = grant.ExpirationPeriodDurationMonth
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t2
				g2.Priority = 1
				g2.Expiration.Count = 1
				g2.Expiration.Duration = grant.ExpirationPeriodDurationHour // This will expire before querying
				g2 = makeGrant(g2)

				g3 := grant2
				g3.ID = "grant-3"
				g3.EffectiveAt = t3
				g3.Priority = 2
				g3.Expiration.Count = 1
				g3.Expiration.Duration = grant.ExpirationPeriodDurationWeek
				g3 = makeGrant(g3)

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2, g3},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 80.0, // due to use before start
								g2.ID: 100.0,
								g3.ID: 100.0,
							},
							Overage: 0,
							At:      start,
						},
						Until: end,
					})

				assert.NoError(t, err)

				assert.NotEmpty(t, res.History)
				assert.Equal(t, 5, len(res.History.Segments()))
				assert.Equal(t, start.In(time.UTC), res.History.Segments()[0].ClosedPeriod.From)
				assert.Equal(t, t1.In(time.UTC), res.History.Segments()[0].ClosedPeriod.To)
				assert.Equal(t, t1.In(time.UTC), res.History.Segments()[1].ClosedPeriod.From)
				assert.Equal(t, t2.In(time.UTC), res.History.Segments()[1].ClosedPeriod.To)
				assert.Equal(t, t2.In(time.UTC), res.History.Segments()[2].ClosedPeriod.From)
				assert.Equal(t, t3.In(time.UTC), res.History.Segments()[2].ClosedPeriod.To)
				assert.Equal(t, t3.In(time.UTC), res.History.Segments()[3].ClosedPeriod.From)
				assert.Equal(t, t2.Add(time.Hour).In(time.UTC), res.History.Segments()[3].ClosedPeriod.To)
				assert.Equal(t, t2.Add(time.Hour).In(time.UTC), res.History.Segments()[4].ClosedPeriod.From)
				assert.Equal(t, end.In(time.UTC), res.History.Segments()[4].ClosedPeriod.To)
			},
		},
		{
			name: "Should use latest aggregation correctly",
			meter: meterpkg.Meter{
				Key:         meterSlug,
				Aggregation: meterpkg.MeterAggregationLatest,
			},
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// burn down with usage after grant effectiveAt
				start := t1
				end := start.Add(time.Hour * 3)

				use(0, start.Add(-time.Minute)) // we need usage so everything's found

				// Grant 1: Active from start, expires during the period
				g1 := makeGrant(grant.Grant{
					ID:          "grant-1-expired",
					Amount:      50.0,
					Priority:    1,
					EffectiveAt: start,
					Expiration: &grant.ExpirationPeriod{
						Duration: grant.ExpirationPeriodDurationHour,
						Count:    1, // Expires after 1 hour
					},
				})

				// Grant 2: Active from start, remains active throughout
				g2 := makeGrant(grant.Grant{
					ID:          "grant-2-active",
					Amount:      100.0,
					Priority:    2,
					EffectiveAt: start,
					Expiration: &grant.ExpirationPeriod{
						Duration: grant.ExpirationPeriodDurationDay,
						Count:    30, // Active for 30 days
					},
				})

				// Grant 3: Becomes effective during the period, remains active
				g3 := makeGrant(grant.Grant{
					ID:          "grant-3-later",
					Amount:      75.0,
					Priority:    3,
					EffectiveAt: start.Add(time.Hour), // Becomes effective after 1 hour
					Expiration: &grant.ExpirationPeriod{
						Duration: grant.ExpirationPeriodDurationDay,
						Count:    30,
					},
				})

				// Grant 4: Future grant, not yet effective
				g4 := makeGrant(grant.Grant{
					ID:          "grant-4-future",
					Amount:      200.0,
					Priority:    4,
					EffectiveAt: start.Add(time.Hour * 4), // Becomes effective after 4 hours (after our calculation period)
					Expiration: &grant.ExpirationPeriod{
						Duration: grant.ExpirationPeriodDurationDay,
						Count:    30,
					},
				})

				// Let's add some usage
				use(15, start)
				use(30, start.Add(time.Hour))
				use(10, start.Add(time.Hour*2))
				use(20, start.Add(time.Hour*3))

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g1, g2, g3, g4},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 50.0,
								g2.ID: 100.0,
								g3.ID: 75.0,
								g4.ID: 200.0,
							},
							Overage: 0,
							At:      start,
						},
						Until: end,
					})
				require.NoError(t, err)

				resJSON, err := json.MarshalIndent(res, "", "  ")
				require.NoError(t, err)

				// Let's start with asserting the ending balance
				// 175 (active total at end) - 10 (last usage value) = 165
				assert.Equal(t, 165.0, res.Snapshot.Balance(), "received following result %s", string(resJSON))

				// Now let's assert the history
				// We should have 2 segments: start -> 1h, 1h -> end
				assert.Equal(t, 2, len(res.History.Segments()), "received following result %s", string(resJSON))

				assert.Equal(t, 15.0, res.History.Segments()[0].TotalUsage)
				assert.Equal(t, 150.0, res.History.Segments()[0].BalanceAtStart.Balance())
				assert.Equal(t, 135.0, res.History.Segments()[0].ApplyUsage().Balance())

				assert.Equal(t, 10.0, res.History.Segments()[1].TotalUsage)
				assert.Equal(t, 175.0, res.History.Segments()[1].BalanceAtStart.Balance())
				assert.Equal(t, 165.0, res.History.Segments()[1].ApplyUsage().Balance())
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()
			streamingConnector := testutils.NewMockStreamingConnector(t)

			mm := lo.Ternary(tc.meter.Key == "", defaultMeter, tc.meter)

			queryFeatureUsage := func(ctx context.Context, from, to time.Time) (float64, error) {
				rows, err := streamingConnector.QueryMeter(ctx, "default", mm, streaming.QueryParams{
					From: &from,
					To:   &to,
				})
				if err != nil {
					return 0.0, err
				}
				if len(rows) > 1 {
					return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
				}
				if len(rows) == 0 {
					return 0.0, nil
				}
				return rows[0].Value, nil
			}
			tc.run(t, engine.NewEngine(engine.EngineConfig{
				QueryUsage: queryFeatureUsage,
			}), func(usage float64, at time.Time) {
				streamingConnector.AddSimpleEvent(meterSlug, usage, at)
			}, mm)
		})
	}
}
