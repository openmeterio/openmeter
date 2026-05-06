package engine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunResult_TotalAvailableGrantAmount(t *testing.T) {
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

	// grant2 := makeGrant(grant.Grant{
	// 	ID:          "grant-2",
	// 	Amount:      100.0,
	// 	Priority:    1,
	// 	EffectiveAt: t1,
	// 	Expiration: &grant.ExpirationPeriod{
	// 		Duration: grant.ExpirationPeriodDurationDay,
	// 		Count:    30,
	// 	},
	// })

	// Tests with single engine
	tt := []struct {
		name  string
		meter meterpkg.Meter
		run   func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter)
	}{
		{
			name: "Should include remaining grant balance",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Past usage so meter is found
				use(1.0, t1.AddDate(-1, 0, 0))

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationDay,
					Count:    30,
				}
				g = makeGrant(g)

				res, err := eng.Run(
					t.Context(),
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
						Until: t1.AddDate(0, 0, 1),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 100.0, res.TotalAvailableGrantAmountAtLastPeriod())
			},
		},
		{
			name: "Should include remaining grant balance and usage of currently inactive grants",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Some usage in period
				use(10.0, t1.Add(time.Hour))

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationDay,
					Count:    30,
				}
				g = makeGrant(g)

				res, err := eng.Run(
					t.Context(),
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
						Until: t1.AddDate(0, 0, 1),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 100.0, res.TotalAvailableGrantAmountAtLastPeriod())
			},
		},
		{
			name: "Should ignore usage if it wasnt covered",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Uncovered usage
				use(100.0, t1.Add(time.Hour))

				res, err := eng.Run(
					t.Context(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{},
							Overage:  0,
							At:       t1,
						},
						Until: t1.AddDate(0, 0, 1),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 0.0, res.TotalAvailableGrantAmountAtLastPeriod())
			},
		},
		{
			name: "Should ignore grants that didnt roll over",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Past usage so meter is found
				use(1.0, t1.AddDate(-1, 0, 0))

				t2 := t1.Add(time.Hour)

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationDay,
					Count:    30,
				}
				g.ResetMaxRollover = 0
				g = makeGrant(g)

				resetTimeline := timeutil.NewSimpleTimeline([]time.Time{t2})

				res, err := eng.Run(
					t.Context(),
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
						Until:  t2,
						Resets: resetTimeline,
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 0.0, res.TotalAvailableGrantAmountAtLastPeriod())
			},
		},
		{
			name: "Should include grant usage up to end of active period of grant",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Usage before expiration
				use(10.0, t1.Add(time.Hour))

				// Usage after expiration
				use(10.0, t1.Add(time.Hour*3))

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationHour,
					Count:    2,
				}
				g = makeGrant(g)

				res, err := eng.Run(
					t.Context(),
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
						Until: t1.AddDate(0, 0, 1),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 10.0, res.TotalAvailableGrantAmountAtLastPeriod())
				// When we have overage this holds true
				assert.Equal(t, res.TotalAvailableGrantAmountAtLastPeriod(), res.Snapshot.Balance()+res.Snapshot.Usage.Usage-res.Snapshot.Overage, "balance %s, usage %s, overage %s", res.Snapshot.Balance(), res.Snapshot.Usage.Usage, res.Snapshot.Overage)
			},
		},
		{
			name: "Should include multi-fold usage of grant recurring during period",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// Usage in first iteration
				use(100.0, t1.AddDate(0, 0, 0).Add(time.Hour))
				// Usage in second iteration
				use(90.0, t1.AddDate(0, 0, 1).Add(time.Hour))
				// Usage in third iteration
				use(80.0, t1.AddDate(0, 0, 2).Add(time.Hour))

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationWeek,
					Count:    2,
				}
				g.Recurrence = &timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodDaily,
					Anchor:   t1,
				}
				g = makeGrant(g)

				res1, err := eng.Run(
					t.Context(),
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
						Until: t1.AddDate(0, 0, 3),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 370.0, res1.TotalAvailableGrantAmountAtLastPeriod())
				// Should be true cause all usage was covered
				assert.Equal(t, res1.TotalAvailableGrantAmountAtLastPeriod(), res1.Snapshot.Balance()+res1.Snapshot.Usage.Usage, "balance %s, usage %s", res1.Snapshot.Balance(), res1.Snapshot.Usage.Usage)

				res2, err := eng.Run(
					t.Context(),
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
						// Going an hour further we're still in the same recurrence iteraton
						Until: t1.AddDate(0, 0, 3).Add(time.Hour),
					},
				)

				require.NoError(t, err)
				assert.Equal(t, 370.0, res2.TotalAvailableGrantAmountAtLastPeriod())
				// Should be true cause all usage was covered
				assert.Equal(t, res2.TotalAvailableGrantAmountAtLastPeriod(), res2.Snapshot.Balance()+res2.Snapshot.Usage.Usage, "balance %s, usage %s", res2.Snapshot.Balance(), res2.Snapshot.Usage.Usage)

				res3, err := eng.Run(
					t.Context(),
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
						// Going an hour back we're still in the same recurrence
						Until: t1.AddDate(0, 0, 3).Add(-time.Hour),
					},
				)

				require.NoError(t, err)
				// So we have 270 used + 20 still available in the period
				assert.Equal(t, 290.0, res3.TotalAvailableGrantAmountAtLastPeriod())
				// Should be true cause all usage was covered
				assert.Equal(t, res3.TotalAvailableGrantAmountAtLastPeriod(), res3.Snapshot.Balance()+res3.Snapshot.Usage.Usage, "balance %s, usage %s", res3.Snapshot.Balance(), res3.Snapshot.Usage.Usage)

				res4, err := eng.Run(
					t.Context(),
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
						Until: t1.AddDate(0, 0, 4),
					},
				)

				require.NoError(t, err)
				// Going an extra day further it's still 280 used up + fresh 100 available as we're on the boundary
				assert.Equal(t, 370.0, res4.TotalAvailableGrantAmountAtLastPeriod())
				// Should be true cause all usage was covered
				assert.Equal(t, res4.TotalAvailableGrantAmountAtLastPeriod(), res4.Snapshot.Balance()+res4.Snapshot.Usage.Usage, "balance %s, usage %s", res4.Snapshot.Balance(), res4.Snapshot.Usage.Usage)
			},
		},
		{
			name: "Should consume overage from previous period without any usage in current period",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc, mm meterpkg.Meter) {
				// usage so meter is found
				use(10.0, t1.AddDate(-1, 0, 0))

				g := grant1
				g.Amount = 100.0
				g.EffectiveAt = t1
				g.Expiration = &grant.ExpirationPeriod{
					Duration: grant.ExpirationPeriodDurationDay,
					Count:    30,
				}
				g = makeGrant(g)

				res, err := eng.Run(
					t.Context(),
					engine.RunParams{
						Meter:  mm,
						Grants: []grant.Grant{g},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g.ID: 100.0,
							},
							At:      t1,
							Overage: 100.0,
						},
						Until: t1.AddDate(0, 0, 1),
					},
				)

				require.NoError(t, err)
				// We had 100 total grants in current period
				assert.Equal(t, 100.0, res.TotalAvailableGrantAmountAtLastPeriod())
				assert.Equal(t, 0.0, res.Snapshot.Balance())
				assert.Equal(t, 0.0, res.Snapshot.Usage.Usage)
				assert.Equal(t, 0.0, res.Snapshot.Overage)
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
