package credit_test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/internal/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestEngine(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)
	meterSlug := "meter-1"

	grant1 := makeGrant(credit.Grant{

		ID:          "grant-1",
		Amount:      100.0,
		Priority:    1,
		EffectiveAt: t1,
		Expiration: credit.ExpirationPeriod{
			Duration: credit.ExpirationPeriodDurationDay,
			Count:    30,
		},
	})

	grant2 := makeGrant(credit.Grant{

		ID:          "grant-2",
		Amount:      100.0,
		Priority:    1,
		EffectiveAt: t1,
		Expiration: credit.ExpirationPeriod{
			Duration: credit.ExpirationPeriodDurationDay,
			Count:    30,
		},
	})

	type addUsageFunc func(usage float64, at time.Time)

	// Tests with single engine
	tt := []struct {
		name string
		run  func(t *testing.T, engine credit.Engine, use addUsageFunc)
	}{
		{
			name: "Should return the same result on subsequent runs",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1 = makeGrant(g1)

				b1, o1, s1, err1 := engine.Run(
					context.Background(),
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1).Add(time.Hour),
					})
				assert.NoError(t, err1)

				b2, o2, s2, err2 := engine.Run(
					context.Background(),
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1).Add(time.Hour),
					})
				assert.NoError(t, err2)

				assert.Equal(t, b1, b2)
				assert.Equal(t, o1, o2)
				assert.Equal(t, s1, s2)
			},
		},
		{
			name: "Reports overage if there are no grants",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				res, overage, segments, err := engine.Run(
					context.Background(),
					[]credit.Grant{},
					credit.GrantBalanceMap{}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 30),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, overage)
				assert.Equal(t, credit.GrantBalanceMap{}, res)
				assert.Equal(t, []credit.GrantBurnDownHistorySegment{
					{
						BalanceAtStart: credit.GrantBalanceMap{},
						GrantUsages:    []credit.GrantUsage{},
						Period: recurrence.Period{
							From: t1,
							To:   t1.AddDate(0, 0, 30),
						},
						TerminationReasons: credit.SegmentTerminationReason{},
						TotalUsage:         50.0,
						Overage:            50.0,
					},
				}, segments)
			},
		},
		{
			name: "Errors if balance was provided for nonexistent grants",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				_, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{},
					credit.GrantBalanceMap{
						grant1.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 30),
					})

				assert.Error(t, err)
			},
		},
		{
			name: "Errors on missing balance for one of the grants",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				g1 := grant1
				g1 = makeGrant(g1)
				g2 := grant2
				g2 = makeGrant(g2)
				_, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						grant1.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 30),
					})

				assert.Error(t, err)
			},
		},
		{
			name: "Able to burn down single active grant",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant1},
					credit.GrantBalanceMap{
						grant1.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 30),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res[grant1.ID])
			},
		},
		{
			name: "Return 0 balance for grant with future effectiveAt",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				grant := grant1
				grant.EffectiveAt = t1.AddDate(0, 0, 10)
				grant = makeGrant(grant)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
			},
		},
		{
			name: "Return 0 balance for deleted grant",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				grant := grant1
				grant.EffectiveAt = t1
				grant.DeletedAt = &t1
				grant = makeGrant(grant)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
			},
		},
		{
			name: "Burns down grant until it's deleted",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				deletionTime := t1.Add(time.Hour)
				use(50.0, deletionTime.Add(-time.Minute))
				use(50.0, deletionTime.Add(time.Minute))
				grant := grant1
				grant.EffectiveAt = t1
				grant.DeletedAt = &deletionTime
				grant = makeGrant(grant)

				res, overage, history, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 50.0, overage) // usage after grant deletion
				assert.Len(t, history, 2)
				assert.Equal(t, 50.0, history[0].TotalUsage)
				assert.Equal(t, 50.0, history[1].TotalUsage)
				assert.Equal(t, 0.0, history[1].BalanceAtStart[grant.ID])
			},
		},
		{
			name: "Burns down grant until it's voided",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				voidingTime := t1.Add(time.Hour)
				use(50.0, voidingTime.Add(-time.Minute))
				use(50.0, voidingTime.Add(time.Minute))
				grant := grant1
				grant.EffectiveAt = t1
				grant.VoidedAt = &voidingTime
				grant = makeGrant(grant)

				res, overage, history, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 50.0, overage) // usage after grant deletion
				assert.Len(t, history, 2)
				assert.Equal(t, 50.0, history[0].TotalUsage)
				assert.Equal(t, 50.0, history[1].TotalUsage)
				assert.Equal(t, 0.0, history[1].BalanceAtStart[grant.ID])
			},
		},
		{
			name: "Return 0 balance for grant with past expiresAt",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				grant := grant1
				grant.EffectiveAt = t1.AddDate(-1, 0, 0)
				grant.Expiration.Duration = credit.ExpirationPeriodDurationDay
				grant.Expiration.Count = 1
				grant = makeGrant(grant)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					}, 0, recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
			},
		},
		{
			name: "Does not burn down grant that expires at start of period",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				grant := grant1
				grant.EffectiveAt = t1.AddDate(-1, 0, 0)
				grant.Expiration.Duration = credit.ExpirationPeriodDurationDay
				grant.Expiration.Count = 1
				grant = makeGrant(grant)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: grant.ExpiresAt,
						To:   grant.ExpiresAt.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
			},
		},
		{
			name: "Burns down grant that becomes active during phase",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(25.0, t1.Add(time.Hour))
				// burn down with usage prior to grant effectiveAt
				use(25.0, t1.Add(-time.Hour))
				grant := grant1
				grant.EffectiveAt = t1
				grant.Expiration.Duration = credit.ExpirationPeriodDurationDay
				grant.Expiration.Count = 30
				grant = makeGrant(grant)

				res, _, segments, err := engine.Run(
					context.Background(),
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 0.0,
					},
					0,
					recurrence.Period{
						From: t1.AddDate(0, 0, -1),
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res[grant1.ID])

				// sets correct starting balance for grant in segment
				assert.Len(t, segments, 2)
				assert.Equal(t, 0.0, segments[0].BalanceAtStart[grant.ID])

				// starting balance doesnt have overage deducted
				assert.Equal(t, grant.Amount, segments[1].BalanceAtStart[grant.ID])
				// but it is stored separately
				assert.Equal(t, 25.0, segments[1].OverageAtStart)
			},
		},
		{
			name: "Burns down multiple grants",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(200, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t1
				g2 = makeGrant(g2)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 0.0, res[grant2.ID])
			},
		},
		{
			name: "Burns down grant with higher priority first",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
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

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g2, g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 80.0, res[grant2.ID])
			},
		},
		{
			name: "Burns down grant that expires first",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t1
				g2.Expiration.Duration = credit.ExpirationPeriodDurationYear
				g2.Expiration.Count = 100
				g2 = makeGrant(g2)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g2, g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 80.0, res[grant2.ID])
			},
		},
		{
			name: "Burns down grant that expires first among many",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// make lots of grants
				nGrant := func(i int) credit.Grant {
					return makeGrant(credit.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      100.0,
						Priority:    1,
						EffectiveAt: t1,
						Expiration: credit.ExpirationPeriod{
							Duration: credit.ExpirationPeriodDurationDay,
							Count:    30,
						},
					})
				}

				numGrants := 100000
				grants := make([]credit.Grant, 0, numGrants)
				for i := 0; i < numGrants; i++ {
					grants = append(grants, nGrant(i))
				}

				// set exp soonner on first grant
				grants[0].Expiration.Count = 29
				grants[0] = makeGrant(grants[0])

				gToBurn := grants[0]
				bm := credit.GrantBalanceMap{}
				for _, g := range grants {
					bm[g.ID] = 100.0
				}

				// shuffle grants
				rand.Shuffle(len(grants), func(i, j int) {
					grants[i], grants[j] = grants[j], grants[i]
				})

				// burn down with usage after grant effectiveAt
				use(99, t1.Add(time.Hour))

				res, _, _, err := engine.Run(
					context.Background(),
					grants,
					bm,
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				for _, g := range grants {
					if g.ID == gToBurn.ID {
						assert.Equal(t, 1.0, res[g.ID])
					} else {
						assert.Equal(t, 100.0, res[g.ID])
					}
				}
			},
		},
		{
			name: "Burns down recurring grant",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1.Recurrence = &recurrence.Recurrence{
					Interval: recurrence.RecurrencePeriodDaily,
					Anchor:   t1,
				}
				g1 = makeGrant(g1)

				res, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1).Add(time.Hour),
					})

				assert.NoError(t, err)
				// grant recurrs daily, so it should be 80.0
				assert.Equal(t, 80.0, res[grant1.ID])
			},
		},
		{
			name: "Burns down recurring grant that takes effect later before it has recurred",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				start := t1
				tg1 := start.AddDate(0, 0, -1)
				tg2 := start.AddDate(0, 0, 1)
				tg2r := start.AddDate(0, 0, 3)

				use(20, start.Add(-time.Hour))                    //g1
				use(20, start.Add(time.Hour))                     //g1
				use(20, start.AddDate(0, 0, 1).Add(-time.Second)) //g1 as its last period
				use(20, start.AddDate(0, 0, 1))                   //g2 due to priority and already effective (only matters before tg2r)

				g1 := grant1
				g1.EffectiveAt = tg1
				g1.Priority = 3
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				g2.Recurrence = &recurrence.Recurrence{
					Interval: recurrence.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				res1, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 10), // right after recurrence
					})

				assert.NoError(t, err)

				assert.Equal(t, 40.0, res1[grant1.ID])
				assert.Equal(t, 100.0, res1[grant2.ID]) // recurred just now (t1 + day10)
			},
		},
		{
			name: "Burns down recurring grant that takes effect later after it has recurred",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				start := t1
				tg1 := start.AddDate(0, 0, -1)
				tg2 := start.AddDate(0, 0, 1)
				tg2r := start.AddDate(0, 0, 3)

				use(20, start.Add(-time.Hour))                    //g1
				use(20, start.Add(time.Hour))                     //g1
				use(20, start.AddDate(0, 0, 1).Add(-time.Second)) //g1 as its last period
				use(20, tg2)                                      //g2 due to priority and already effective (only matters before tg2r)
				use(20, tg2r)                                     //g2 after first recurrence

				g1 := grant1
				g1.EffectiveAt = tg1
				g1.Priority = 3
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				g2.Recurrence = &recurrence.Recurrence{
					Interval: recurrence.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				res2, _, _, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 10).Add(-time.Hour), // right before recurrence
					})

				assert.NoError(t, err)

				assert.Equal(t, 40.0, res2[grant1.ID])
				// 20 usage as above
				assert.Equal(t, 80.0, res2[grant2.ID])
			},
		},
		{
			name: "Return GrantBurnDownHistory",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
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
				g1.Expiration.Duration = credit.ExpirationPeriodDurationMonth
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = tg2
				g2.Priority = 1
				// so they dont expire
				g2.Expiration.Count = 2
				g2.Expiration.Duration = credit.ExpirationPeriodDurationMonth
				g2.Recurrence = &recurrence.Recurrence{
					Interval: recurrence.RecurrencePeriodWeek,
					Anchor:   tg2r,
				}
				g2 = makeGrant(g2)

				_, _, segments, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: start,
						To:   end,
					})

				assert.NoError(t, err)

				assert.NotEmpty(t, segments)
				assert.Equal(t, 4, len(segments))
			},
		},
		{
			name: "Should calculate sequential periods accross timezones and convert to UTC",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
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
				g1.Expiration.Duration = credit.ExpirationPeriodDurationMonth
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = t2
				g2.Priority = 1
				g2.Expiration.Count = 1
				g2.Expiration.Duration = credit.ExpirationPeriodDurationHour // This will expire before querying
				g2 = makeGrant(g2)

				g3 := grant2
				g3.ID = "grant-3"
				g3.EffectiveAt = t3
				g3.Priority = 2
				g3.Expiration.Count = 1
				g3.Expiration.Duration = credit.ExpirationPeriodDurationWeek
				g3 = makeGrant(g3)

				_, _, segments, err := engine.Run(
					context.Background(),
					[]credit.Grant{g1, g2, g3},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
						g3.ID: 100.0,
					},
					0,
					recurrence.Period{
						From: start,
						To:   end,
					})

				assert.NoError(t, err)

				assert.NotEmpty(t, segments)
				assert.Equal(t, 5, len(segments))
				assert.Equal(t, start.In(time.UTC), segments[0].Period.From)
				assert.Equal(t, t1.In(time.UTC), segments[0].Period.To)
				assert.Equal(t, t1.In(time.UTC), segments[1].Period.From)
				assert.Equal(t, t2.In(time.UTC), segments[1].Period.To)
				assert.Equal(t, t2.In(time.UTC), segments[2].Period.From)
				assert.Equal(t, t3.In(time.UTC), segments[2].Period.To)
				assert.Equal(t, t3.In(time.UTC), segments[3].Period.From)
				assert.Equal(t, t2.Add(time.Hour).In(time.UTC), segments[3].Period.To)
				assert.Equal(t, t2.Add(time.Hour).In(time.UTC), segments[4].Period.From)
				assert.Equal(t, end.In(time.UTC), segments[4].Period.To)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			streamingConnector := testutils.NewMockStreamingConnector(t)

			queryFeatureUsage := func(ctx context.Context, from, to time.Time) (float64, error) {
				rows, err := streamingConnector.QueryMeter(ctx, "default", meterSlug, &streaming.QueryParams{
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
			tc.run(t, credit.NewEngine(queryFeatureUsage, models.WindowSizeMinute), func(usage float64, at time.Time) {
				streamingConnector.AddSimpleEvent(meterSlug, usage, at)
			})
		})
	}

	// tests with multiple engines and fuzzing
	tt2 := []struct {
		name   string
		repeat int
		run    func(t *testing.T, queryFn func(ctx context.Context, from time.Time, to time.Time) (float64, error), use addUsageFunc)
	}{
		{
			name:   "Calculating same period in 2 runs yields same result as calculations in one run",
			repeat: 1,
			run: func(t *testing.T, queryFn func(ctx context.Context, from time.Time, to time.Time) (float64, error), use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				start := t1

				use(2000, start.Add(time.Hour))

				intermediate := start.Add(time.Hour * 3)
				end := start.Add(time.Hour * 6)

				g1 := grant1
				g1.EffectiveAt = start
				g1.Priority = 5
				g1 = makeGrant(g1)

				g2 := grant2
				g2.EffectiveAt = intermediate.Add(time.Minute * 1)
				g2.Priority = 2
				g2 = makeGrant(g2)

				startingBalance := credit.GrantBalanceMap{
					g1.ID: 100.0,
					g2.ID: 82.0,
				}

				engine1 := credit.NewEngine(queryFn, models.WindowSizeMinute) // runs for first part
				engine2 := credit.NewEngine(queryFn, models.WindowSizeMinute) // runs for second part
				engine3 := credit.NewEngine(queryFn, models.WindowSizeMinute) // runs for both parts

				intermediateBalance, overage, _, err := engine1.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					startingBalance,
					0,
					recurrence.Period{
						From: start,
						To:   intermediate,
					})

				assert.NoError(t, err)

				finalBalance1, _, _, err := engine2.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					intermediateBalance,
					overage,
					recurrence.Period{
						From: intermediate,
						To:   end,
					})

				assert.NoError(t, err)

				finalBalance2, _, _, err := engine3.Run(
					context.Background(),
					[]credit.Grant{g1, g2},
					startingBalance,
					0,
					recurrence.Period{
						From: start,
						To:   end,
					})

				assert.NoError(t, err)

				// assert equivalence
				assert.Equal(t, finalBalance1, finalBalance2)
			},
		},
		{
			name:   "Deterministic",
			repeat: 10,
			run: func(t *testing.T, queryFn func(ctx context.Context, from time.Time, to time.Time) (float64, error), use addUsageFunc) {
				granularity := time.Minute

				// run for 1 month
				start := t1.Truncate(granularity)
				end := start.AddDate(0, 1, 0).Truncate(granularity)

				// params
				numOfRuns := rand.Intn(19) + 1
				numOfGrants := rand.Intn(19) + 1
				numOfUsageEvents := rand.Intn(9999) + 1

				// create random grants
				grants := make([]credit.Grant, numOfGrants)
				for i := 0; i < numOfGrants; i++ {
					grant := credit.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:    gofakeit.Uint8(),
						EffectiveAt: gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: credit.ExpirationPeriod{
							Duration: credit.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint8(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &recurrence.Recurrence{
							Interval: recurrence.RecurrencePeriodDaily,
							Anchor:   gofakeit.DateRange(start, end).Truncate(granularity),
						}
					}
					grants[i] = makeGrant(grant)
				}

				// ingest usage events
				for i := 0; i < numOfUsageEvents; i++ {
					use(float64(gofakeit.IntRange(1, 100)), gofakeit.DateRange(start, end).Truncate(granularity))
				}

				// configure starting balances
				startingBalances := make(credit.GrantBalanceMap)
				for _, grant := range grants {
					startingBalances[grant.ID] = float64(gofakeit.IntRange(1, int(grant.Amount)))
				}

				// run calculation multiple times
				balances := startingBalances.Copy()
				results := make([]credit.GrantBalanceMap, numOfRuns)
				for i := 0; i < numOfRuns; i++ {
					engine := credit.NewEngine(queryFn, models.WindowSizeMinute)
					gCp := make([]credit.Grant, len(grants))
					copy(gCp, grants)
					result, _, _, err := engine.Run(
						context.Background(),
						gCp, balances, 0,
						recurrence.Period{
							From: start,
							To:   end,
						})
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					results[i] = result.Copy()
				}

				sumVals := func(m credit.GrantBalanceMap) float64 {
					sum := 0.0
					for _, v := range m {
						sum += v
					}
					return sum
				}

				for _, result := range results {
					assert.Equal(t, sumVals(results[0]), sumVals(result))
					assert.Equal(t, results[0], result)
				}
			},
		},
		{
			name:   "Fuzzing sequences",
			repeat: 10,
			run: func(t *testing.T, queryFn func(ctx context.Context, from time.Time, to time.Time) (float64, error), use addUsageFunc) {
				granularity := time.Minute

				// run for 1 month
				start := t1.Truncate(granularity)
				end := start.AddDate(0, 1, 0).Truncate(granularity)

				// fuzz params
				numOfEngines := rand.Intn(19) + 1
				numOfGrants := rand.Intn(19) + 1
				numOfUsageEvents := rand.Intn(99) + 1

				// create random grants
				grants := make([]credit.Grant, numOfGrants)
				for i := 0; i < numOfGrants; i++ {
					grant := credit.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:    gofakeit.Uint8(),
						EffectiveAt: gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: credit.ExpirationPeriod{
							Duration: credit.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint8(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &recurrence.Recurrence{
							Interval: recurrence.RecurrencePeriodDaily,
							Anchor:   gofakeit.DateRange(start, end).Truncate(granularity),
						}
					}
					grants[i] = makeGrant(grant)
				}

				// ingest usage events
				for i := 0; i < numOfUsageEvents; i++ {
					use(float64(gofakeit.IntRange(100, 1000)), gofakeit.DateRange(start.Add(time.Hour), end.Add(-time.Hour)).Truncate(granularity))
				}

				// configure starting balances
				startingBalances := make(credit.GrantBalanceMap)
				for _, grant := range grants {
					startingBalances[grant.ID] = float64(gofakeit.IntRange(1, int(grant.Amount)))
				}

				// run calculation on single engine
				singleEngine := credit.NewEngine(queryFn, models.WindowSizeMinute)
				gCp := make([]credit.Grant, len(grants))
				copy(gCp, grants)
				singleEngineResult, _, _, err := singleEngine.Run(
					context.Background(),
					gCp, startingBalances, 0,
					recurrence.Period{
						From: start,
						To:   end,
					})
				if err != nil {
					// lets save ourselves the calculation if this already fails
					t.Fatalf("unexpected error: %v", err)
				}

				// run calculation on multiple engines
				balances := startingBalances.Copy()
				pStart := start.Truncate(granularity)

				runLength := end.Sub(start) / time.Duration(numOfEngines)
				overage := 0.0

				// periods := make([]recurrence.Period, 0, numOfEngines)

				for i := 0; i < numOfEngines; i++ {
					// get period end by even distribution
					pEnd := pStart.Add(runLength).Truncate(granularity)
					if i == numOfEngines-1 {
						pEnd = end
					}

					// periods = append(periods, recurrence.Period{
					// 	From: pStart,
					// 	To:   pEnd,
					// })

					engine := credit.NewEngine(queryFn, models.WindowSizeMinute)
					gCp := make([]credit.Grant, len(grants))
					copy(gCp, grants)
					balances, overage, _, err = engine.Run(
						context.Background(),
						gCp, balances, overage,
						recurrence.Period{
							From: pStart,
							To:   pEnd,
						})
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					pStart = pEnd
				}

				assert.Equal(t, singleEngineResult, balances)
			},
		},
	}

	for _, tc := range tt2 {
		for i := 0; i < int(math.Min(float64(tc.repeat), 1.0)); i++ {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				streamingConnector := testutils.NewMockStreamingConnector(t)

				queryFeatureUsage := func(ctx context.Context, from, to time.Time) (float64, error) {
					rows, err := streamingConnector.QueryMeter(ctx, "default", meterSlug, &streaming.QueryParams{
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
				tc.run(t, queryFeatureUsage, func(usage float64, at time.Time) {
					streamingConnector.AddSimpleEvent(meterSlug, usage, at)
				})
			})
		}
	}
}

func makeGrant(grant credit.Grant) credit.Grant {
	grant.ExpiresAt = grant.GetExpiration()
	return grant
}
