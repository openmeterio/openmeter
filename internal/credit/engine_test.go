package credit_test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/testutils"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/stretchr/testify/assert"
)

func TestEngine(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)
	farInPast := t1.AddDate(-10, 0, 0)
	meterSlug := "meter-1"

	grant1 := makeGrant(credit.Grant{

		ID:            "grant-1",
		EntitlementID: "entitlement-1",
		Amount:        100.0,
		Priority:      1,
		EffectiveAt:   t1,
		Expiration: credit.ExpirationPeriod{
			Duration: credit.ExpirationPeriodDurationDay,
			Count:    30,
		},
		// ExpiresAt: time.Now().AddDate(0, 0, 30),
		// ResetMaxRollover: 1,
		// Recurrence: credit.Recurrence{
		// 	MaxRolloverAmount: 1,
		// 	Period:            credit.RecurrencePeriodMonthly,
		// 	Anchor:            time.Now().AddDate(0, 0, 0),
		// },
	})

	grant2 := makeGrant(credit.Grant{

		ID:            "grant-2",
		EntitlementID: "entitlement-1",
		Amount:        100.0,
		Priority:      1,
		EffectiveAt:   t1,
		Expiration: credit.ExpirationPeriod{
			Duration: credit.ExpirationPeriodDurationDay,
			Count:    30,
		},
		// ExpiresAt: time.Now().AddDate(0, 0, 30),
		// ResetMaxRollover: 1,
		// Recurrence: credit.Recurrence{
		// 	MaxRolloverAmount: 1,
		// 	Period:            credit.RecurrencePeriodMonthly,
		// 	Anchor:            time.Now().AddDate(0, 0, 0),
		// },
	})

	type addUsageFunc func(usage float64, at time.Time)

	// Tests with single engine
	tt := []struct {
		name string
		run  func(t *testing.T, engine credit.Engine, use addUsageFunc)
	}{
		{
			name: "Should error if already run",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1 = makeGrant(g1)

				engine.Run(
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					credit.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1).Add(time.Hour),
					})

				_, _, _, err := engine.Run(
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					credit.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1).Add(time.Hour),
					})

				assert.Error(t, err, "engine has already run")
			},
		},
		{
			name: "Able to burn down single active grant",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				use(50.0, t1.Add(time.Hour))
				res, _, _, err := engine.Run(
					[]credit.Grant{grant1},
					credit.GrantBalanceMap{
						grant1.ID: 100.0,
					}, 0, credit.Period{
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
					[]credit.Grant{grant},
					credit.GrantBalanceMap{}, 0, credit.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 5),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
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
					[]credit.Grant{grant},
					credit.GrantBalanceMap{}, 0, credit.Period{
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
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					},
					0,
					credit.Period{
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

				res, _, _, err := engine.Run(
					[]credit.Grant{grant},
					credit.GrantBalanceMap{
						grant.ID: 100.0,
					},
					0,
					credit.Period{
						From: t1.AddDate(0, 0, -1),
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 50.0, res[grant1.ID])
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
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					credit.Period{
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
					[]credit.Grant{g2, g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					credit.Period{
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
					[]credit.Grant{g2, g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
						g2.ID: 100.0,
					},
					0,
					credit.Period{
						From: t1,
						To:   t1.AddDate(0, 0, 1),
					})

				assert.NoError(t, err)
				assert.Equal(t, 0.0, res[grant1.ID])
				assert.Equal(t, 80.0, res[grant2.ID])
			},
		},
		{
			name: "Burns down recurring grant",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				// burn down with usage after grant effectiveAt
				use(120, t1.Add(time.Hour))
				g1 := grant1
				g1.EffectiveAt = t1
				g1.Recurrence = &credit.Recurrence{
					Period:            credit.RecurrencePeriodDaily,
					Anchor:            t1,
					MaxRolloverAmount: -1, // currently unused
				}
				g1 = makeGrant(g1)

				res, _, _, err := engine.Run(
					[]credit.Grant{g1},
					credit.GrantBalanceMap{
						g1.ID: 100.0,
					},
					0,
					credit.Period{
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
				g2.Recurrence = &credit.Recurrence{
					Period:            credit.RecurrencePeriodWeekly,
					Anchor:            tg2r,
					MaxRolloverAmount: -1, // currently unused
				}
				g2 = makeGrant(g2)

				res1, _, _, err := engine.Run(
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					credit.Period{
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
				g2.Recurrence = &credit.Recurrence{
					Period:            credit.RecurrencePeriodWeekly,
					Anchor:            tg2r,
					MaxRolloverAmount: -1, // currently unused
				}
				g2 = makeGrant(g2)

				res2, _, _, err := engine.Run(
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					credit.Period{
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
				g2.Recurrence = &credit.Recurrence{
					Period:            credit.RecurrencePeriodWeekly,
					Anchor:            tg2r,
					MaxRolloverAmount: -1, // currently unused
				}
				g2 = makeGrant(g2)

				_, _, segments, err := engine.Run(
					[]credit.Grant{g1, g2},
					credit.GrantBalanceMap{
						g1.ID: 80.0, // due to use before start
						g2.ID: 100.0,
					},
					0,
					credit.Period{
						From: start,
						To:   end,
					})

				assert.NoError(t, err)

				assert.NotEmpty(t, segments)
				assert.Equal(t, 4, len(segments))
			},
		},
		{
			name: "Test windowing",
			run: func(t *testing.T, engine credit.Engine, use addUsageFunc) {
				t.Skip(`
                    Windowing is not inherently part of the engine, its a property of the persistence layer.
                    TODO: how to test this.
                `)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// TODO: use a different mock
			streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: farInPast})

			queryFeatureUsage := func(from, to time.Time) (float64, error) {
				rows, err := streamingConnector.QueryMeter(context.TODO(), "default", meterSlug, &streaming.QueryParams{
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
			tc.run(t, credit.NewEngine(queryFeatureUsage), func(usage float64, at time.Time) {
				streamingConnector.AddSimpleEvent(meterSlug, usage, at)
			})
		})
	}

	// tests with multiple engines and fuzzing
	tt2 := []struct {
		name   string
		repeat int
		run    func(t *testing.T, queryFn func(from time.Time, to time.Time) (float64, error), use addUsageFunc)
	}{
		{
			name:   "Calculating same period in 2 runs yields same result as calculatin in one run",
			repeat: 1,
			run: func(t *testing.T, queryFn func(from time.Time, to time.Time) (float64, error), use addUsageFunc) {
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

				engine1 := credit.NewEngine(queryFn) // runs for first part
				engine2 := credit.NewEngine(queryFn) // runs for second part
				engine3 := credit.NewEngine(queryFn) // runs for both parts

				intermediateBalance, overage, _, err := engine1.Run(
					[]credit.Grant{g1, g2},
					startingBalance,
					0,
					credit.Period{
						From: start,
						To:   intermediate,
					})

				assert.NoError(t, err)

				finalBalance1, _, _, err := engine2.Run(
					[]credit.Grant{g1, g2},
					intermediateBalance,
					overage,
					credit.Period{
						From: intermediate,
						To:   end,
					})

				assert.NoError(t, err)

				finalBalance2, _, _, err := engine3.Run(
					[]credit.Grant{g1, g2},
					startingBalance,
					0,
					credit.Period{
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
			run: func(t *testing.T, queryFn func(from time.Time, to time.Time) (float64, error), use addUsageFunc) {
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
						ID:            credit.GrantID(fmt.Sprintf("grant-%d", i)),
						EntitlementID: entitlement.EntitlementID("entitlement-1"),
						Amount:        float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:      gofakeit.Uint8(),
						EffectiveAt:   gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: credit.ExpirationPeriod{
							Duration: credit.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint8(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &credit.Recurrence{
							Period:            credit.RecurrencePeriodDaily,
							Anchor:            gofakeit.DateRange(start, end).Truncate(granularity),
							MaxRolloverAmount: -1, // currently unused
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
					engine := credit.NewEngine(queryFn)
					gCp := make([]credit.Grant, len(grants))
					copy(gCp, grants)
					result, _, _, err := engine.Run(gCp, balances, 0, credit.Period{
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
			run: func(t *testing.T, queryFn func(from time.Time, to time.Time) (float64, error), use addUsageFunc) {
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
						ID:            credit.GrantID(fmt.Sprintf("grant-%d", i)),
						EntitlementID: entitlement.EntitlementID("entitlement-1"),
						Amount:        float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:      gofakeit.Uint8(),
						EffectiveAt:   gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: credit.ExpirationPeriod{
							Duration: credit.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint8(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &credit.Recurrence{
							Period:            credit.RecurrencePeriodDaily,
							Anchor:            gofakeit.DateRange(start, end).Truncate(granularity),
							MaxRolloverAmount: -1, // currently unused
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
				singleEngine := credit.NewEngine(queryFn)
				gCp := make([]credit.Grant, len(grants))
				copy(gCp, grants)
				singleEngineResult, _, _, err := singleEngine.Run(gCp, startingBalances, 0, credit.Period{
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

				// periods := make([]credit.Period, 0, numOfEngines)

				for i := 0; i < numOfEngines; i++ {
					// get period end by even distribution
					pEnd := pStart.Add(runLength).Truncate(granularity)
					if i == numOfEngines-1 {
						pEnd = end
					}

					// periods = append(periods, credit.Period{
					// 	From: pStart,
					// 	To:   pEnd,
					// })

					engine := credit.NewEngine(queryFn)
					gCp := make([]credit.Grant, len(grants))
					copy(gCp, grants)
					balances, overage, _, err = engine.Run(gCp, balances, overage, credit.Period{
						From: pStart,
						To:   pEnd,
					})
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					pStart = pEnd
				}

				assert.Equal(t, singleEngineResult, balances)
				// t.Log(numOfEngines, numOfGrants, numOfUsageEvents, runLength.String())
				// t.Logf("\n")
				// t.Log(slicesx.Map(grants, func(g credit.Grant) string {
				// 	return fmt.Sprintf("%s: %v", g.ID, g.Recurrence)
				// }))
				// t.Logf("\n")
				// t.Log(periods)
			},
		},
	}

	for _, tc := range tt2 {
		for i := 0; i < int(math.Min(float64(tc.repeat), 1.0)); i++ {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				// TODO: use a different mock
				streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: farInPast})

				queryFeatureUsage := func(from, to time.Time) (float64, error) {
					rows, err := streamingConnector.QueryMeter(context.TODO(), "default", meterSlug, &streaming.QueryParams{
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
