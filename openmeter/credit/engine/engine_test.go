package engine_test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func Test_Fuzzing(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)
	meterSlug := "meter-1"

	meter := meterpkg.Meter{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: "default",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:         meterSlug,
		EventType:   "requests",
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

				startingBalance := balance.Map{
					g1.ID: 100.0,
					g2.ID: 82.0,
				}

				engine1 := engine.NewEngine(engine.EngineConfig{
					QueryUsage: queryFn,
				}) // runs for first part
				engine2 := engine.NewEngine(engine.EngineConfig{
					QueryUsage: queryFn,
				}) // runs for second part
				engine3 := engine.NewEngine(engine.EngineConfig{
					QueryUsage: queryFn,
				}) // runs for both parts

				res, err := engine1.Run(
					context.Background(),
					engine.RunParams{
						Grants: []grant.Grant{g1, g2},
						StartingSnapshot: balance.Snapshot{
							Balances: startingBalance,
							Overage:  0,
							At:       start,
						},
						Meter: meter,
						Until: intermediate,
					})

				assert.NoError(t, err)

				res2, err := engine2.Run(
					context.Background(),
					engine.RunParams{
						Grants:           []grant.Grant{g1, g2},
						StartingSnapshot: res.Snapshot,
						Until:            end,
					})

				assert.NoError(t, err)

				res3, err := engine3.Run(
					context.Background(),
					engine.RunParams{
						Grants:           []grant.Grant{g1, g2},
						StartingSnapshot: res.Snapshot,
						Until:            end,
						Meter:            meter,
					})

				assert.NoError(t, err)

				// assert equivalence
				assert.Equal(t, res2.Snapshot.Balances, res3.Snapshot.Balances)
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
				grants := make([]grant.Grant, numOfGrants)
				for i := 0; i < numOfGrants; i++ {
					grant := grant.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:    gofakeit.Uint8(),
						EffectiveAt: gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: &grant.ExpirationPeriod{
							Duration: grant.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint32(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &timeutil.Recurrence{
							Interval: timeutil.RecurrencePeriodDaily,
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
				startingBalances := make(balance.Map)
				for _, grant := range grants {
					startingBalances[grant.ID] = float64(gofakeit.IntRange(1, int(grant.Amount)))
				}

				// run calculation multiple times
				balances := startingBalances.Clone()
				results := make([]balance.Map, numOfRuns)
				for i := 0; i < numOfRuns; i++ {
					eng := engine.NewEngine(engine.EngineConfig{
						QueryUsage: queryFn,
					})
					gCp := make([]grant.Grant, len(grants))
					copy(gCp, grants)
					result, err := eng.Run(
						context.Background(),
						engine.RunParams{
							Grants: gCp,
							StartingSnapshot: balance.Snapshot{
								Balances: balances,
								Overage:  0,
								At:       start,
							},
							Meter: meter,
							Until: end,
						})
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					results[i] = result.Snapshot.Balances.Clone()
				}

				sumVals := func(m balance.Map) float64 {
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
				grants := make([]grant.Grant, numOfGrants)
				for i := 0; i < numOfGrants; i++ {
					grant := grant.Grant{
						ID:          fmt.Sprintf("grant-%d", i),
						Amount:      float64(gofakeit.IntRange(10000, 1000000)), // input value limited to ints
						Priority:    gofakeit.Uint8(),
						EffectiveAt: gofakeit.DateRange(start, end).Truncate(granularity),
						Expiration: &grant.ExpirationPeriod{
							Duration: grant.ExpirationPeriodDurationDay,
							Count:    gofakeit.Uint32(),
						},
					}

					if gofakeit.Bool() {
						grant.Recurrence = &timeutil.Recurrence{
							Interval: timeutil.RecurrencePeriodDaily,
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
				startingBalances := make(balance.Map)
				for _, grant := range grants {
					startingBalances[grant.ID] = float64(gofakeit.IntRange(1, int(grant.Amount)))
				}

				// run calculation on single engine
				singleEngine := engine.NewEngine(engine.EngineConfig{
					QueryUsage: queryFn,
				})
				gCp := make([]grant.Grant, len(grants))
				copy(gCp, grants)
				singleEngineResult, err := singleEngine.Run(
					context.Background(),
					engine.RunParams{
						Grants: gCp,
						StartingSnapshot: balance.Snapshot{
							Balances: startingBalances,
							Overage:  0,
							At:       start,
						},
						Meter: meter,
						Until: end,
					})
				if err != nil {
					// lets save ourselves the calculation if this already fails
					t.Fatalf("unexpected error: %v", err)
				}

				// run calculation on multiple engines
				balances := startingBalances.Clone()
				pStart := start.Truncate(granularity)

				runLength := end.Sub(start) / time.Duration(numOfEngines)
				overage := 0.0

				// periods := make([]timeutil.Period, 0, numOfEngines)

				for i := 0; i < numOfEngines; i++ {
					// get period end by even distribution
					pEnd := pStart.Add(runLength).Truncate(granularity)
					if i == numOfEngines-1 {
						pEnd = end
					}

					eng := engine.NewEngine(engine.EngineConfig{
						QueryUsage: queryFn,
					})
					gCp := make([]grant.Grant, len(grants))
					copy(gCp, grants)
					res, err := eng.Run(
						context.Background(),
						engine.RunParams{
							Grants: gCp,
							StartingSnapshot: balance.Snapshot{
								Balances: balances,
								Overage:  overage,
								At:       pStart,
							},
							Until: pEnd,
							Meter: meter,
						})
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					balances = res.Snapshot.Balances
					overage = res.Snapshot.Overage

					pStart = pEnd
				}

				assert.Equal(t, singleEngineResult.Snapshot.Balances, balances)
			},
		},
	}

	for _, tc := range tt2 {
		for i := 0; i < int(math.Min(float64(tc.repeat), 1.0)); i++ {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				streamingConnector := testutils.NewMockStreamingConnector(t)

				queryFeatureUsage := func(ctx context.Context, from, to time.Time) (float64, error) {
					rows, err := streamingConnector.QueryMeter(ctx, "default", meter, streaming.QueryParams{
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

func makeGrant(grant grant.Grant) grant.Grant {
	grant.ExpiresAt = grant.GetExpiration()
	return grant
}

type addUsageFunc func(usage float64, at time.Time)
