package engine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestReset(t *testing.T) {
	t1 := testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")
	meterSlug := "meter-1"

	meter := meterpkg.Meter{
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

	setup := func(t *testing.T) (engine.Engine, addUsageFunc) {
		streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

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

		return engine.NewEngine(engine.EngineConfig{
				QueryUsage: queryFeatureUsage,
			}), func(usage float64, at time.Time) {
				streamingConnector.AddSimpleEvent(meterSlug, usage, at)
			}
	}

	t.Run("Should reset roll over grant balance after one reset", func(t *testing.T) {
		eng, use := setup(t)
		use(10.0, t1.Add(time.Hour))

		g1 := grant1
		g1.ResetMaxRollover = 50.0

		// grants can roll over their own balance so we don't check the details of that
		res, err := eng.Run(
			context.Background(),
			engine.RunParams{
				Meter:  meter,
				Grants: []grant.Grant{g1},
				StartingSnapshot: balance.Snapshot{
					Usage: balance.SnapshottedUsage{
						Since: t1.AddDate(0, 0, -1), // Last "reset time", outside this period, arbitrary
						Usage: 0.0,
					},
					Balances: balance.Map{
						g1.ID: 100.0,
					},
					Overage: 0,
					At:      t1,
				},
				Until: t1.AddDate(0, 0, 1),
				ResetBehavior: grant.ResetBehavior{
					PreserveOverage: false,
				},
				Resets: timeutil.NewSimpleTimeline([]time.Time{t1.Add(time.Hour * 5)}),
			},
		)
		assert.NoError(t, err)

		// The grant should be rolled over:
		// 100 - 10 = 90;
		// Min(50, max(0, 90)) = 50
		assert.Equal(t, 50.0, res.Snapshot.Balances[grant1.ID])

		// Usage since last reset should be captured in snapshot
		assert.Equal(t, 0.0, res.Snapshot.Usage.Usage)                 // 0 usage after 5h mark
		assert.Equal(t, t1.Add(time.Hour*5), res.Snapshot.Usage.Since) // should mark since the last reset time

		// History should have 2 segments, one before and one after the reset
		assert.Equal(t, 2, len(res.History.Segments()))

		// The first segment should have a balance of 100 with 10 usage
		assert.Equal(t, 100.0, res.History.Segments()[0].BalanceAtStart.Balance())
		assert.Equal(t, 0.0, res.History.Segments()[0].OverageAtStart)
		assert.Equal(t, 10.0, res.History.Segments()[0].TotalUsage)

		// It should end with a reset
		assert.True(t, res.History.Segments()[0].TerminationReasons.UsageReset)

		// The second segment should have a balance of 50 with no usage
		assert.Equal(t, 50.0, res.History.Segments()[1].BalanceAtStart.Balance())
		assert.Equal(t, 0.0, res.History.Segments()[1].OverageAtStart)
		assert.Equal(t, 0.0, res.History.Segments()[1].TotalUsage)
	})

	t.Run("Should carry over overage to next period", func(t *testing.T) {
		eng, use := setup(t)
		use(10.0, t1.Add(time.Hour))
		use(100.0, t1.Add(time.Hour*2))

		g1 := grant1
		g1.ResetMaxRollover = 50.0
		g1.ResetMinRollover = 50.0

		// grants can roll over their own balance so we don't check the details of that
		res, err := eng.Run(
			context.Background(),
			engine.RunParams{
				Meter:  meter,
				Grants: []grant.Grant{g1},
				StartingSnapshot: balance.Snapshot{
					Balances: balance.Map{
						g1.ID: 100.0,
					},
					Overage: 0,
					At:      t1,
				},
				Until: t1.AddDate(0, 0, 1),
				ResetBehavior: grant.ResetBehavior{
					PreserveOverage: true,
				},
				Resets: timeutil.NewSimpleTimeline([]time.Time{t1.Add(time.Hour * 5)}),
			},
		)
		assert.NoError(t, err)

		// The grant should be rolled over:
		assert.Equal(t, 40.0, res.Snapshot.Balances[grant1.ID])

		// History should have 2 segments, one before and one after the reset
		assert.Equal(t, 2, len(res.History.Segments()))

		// The first segment should have a balance of 0 with 10 overage
		assert.Equal(t, 100.0, res.History.Segments()[0].BalanceAtStart.Balance())
		assert.Equal(t, 0.0, res.History.Segments()[0].OverageAtStart)
		assert.Equal(t, 10.0, res.History.Segments()[0].Overage)
		assert.Equal(t, 110.0, res.History.Segments()[0].TotalUsage)

		// It should end with a reset
		assert.True(t, res.History.Segments()[0].TerminationReasons.UsageReset)

		// The second segment should have a balance of 50 - 10 with no usage (minRolloverAmount + overage)
		assert.Equal(t, 40.0, res.History.Segments()[1].BalanceAtStart.Balance())
		assert.Equal(t, 0.0, res.History.Segments()[1].OverageAtStart)
		assert.Equal(t, 0.0, res.History.Segments()[1].TotalUsage)
	})

	t.Run("No reset", func(t *testing.T) {
		eng, use := setup(t)
		use(10.0, t1.Add(time.Hour))

		g2 := grant1
		g2.EffectiveAt = t1.Add(time.Hour * 2)

		u := balance.SnapshottedUsage{
			Since: t1.AddDate(0, 0, -1),
			Usage: 10.0,
		}

		res, err := eng.Run(
			context.Background(),
			engine.RunParams{
				Meter:  meter,
				Grants: []grant.Grant{grant1, g2},
				StartingSnapshot: balance.Snapshot{
					Usage: u,
					Balances: balance.Map{
						grant1.ID: 100.0,
						g2.ID:     100.0,
					},
					Overage: 0,
					At:      t1,
				},
				Until: t1.AddDate(0, 0, 1),
			},
		)
		assert.NoError(t, err)

		// If there was no reset, should extend the starting snapshot with the current usage data
		assert.Equal(t, 20.0, res.Snapshot.Usage.Usage) // 10 + 10
		assert.Equal(t, u.Since, res.Snapshot.Usage.Since)

		// Should have 2 periods, start - g2, g2 - end
		assert.Equal(t, 2, len(res.History.Segments()))

		assert.False(t, res.History.Segments()[0].TerminationReasons.UsageReset)
		assert.False(t, res.History.Segments()[1].TerminationReasons.UsageReset)
	})

	t.Run("Should return starting balance if the end of the queried period is a reset", func(t *testing.T) {
		eng, use := setup(t)
		use(10.0, t1.Add(time.Hour))

		g1 := grant1
		g1.ResetMaxRollover = 50.0

		resetTime := t1.AddDate(0, 0, 1)

		res, err := eng.Run(
			context.Background(),
			engine.RunParams{
				Meter:  meter,
				Grants: []grant.Grant{g1},
				StartingSnapshot: balance.Snapshot{
					Balances: balance.Map{
						g1.ID: 100.0,
					},
					Overage: 0,
					At:      t1,
				},
				Until:  resetTime,
				Resets: timeutil.NewSimpleTimeline([]time.Time{resetTime}),
			},
		)
		assert.NoError(t, err)

		// There cannot be any usage right at reset
		assert.Equal(t, 0.0, res.Snapshot.Usage.Usage)
		assert.Equal(t, resetTime, res.Snapshot.Usage.Since)

		// Should have 2 periods, start - reset, reset - end where reset = end, 2nd period is 0 length
		assert.Equal(t, 2, len(res.History.Segments()), "expected: %+v, got %+v, history: %+v", 2, len(res.History.Segments()), res.History.Segments())

		assert.True(t, res.History.Segments()[0].TerminationReasons.UsageReset)
		assert.False(t, res.History.Segments()[1].TerminationReasons.UsageReset)
	})

	t.Run("Should error if a reset is provided for the starting snapshot", func(t *testing.T) {
		eng, use := setup(t)
		use(10.0, t1.Add(time.Hour))

		g1 := grant1
		g1.ResetMaxRollover = 50.0

		_, err := eng.Run(
			context.Background(),
			engine.RunParams{
				Meter:  meter,
				Grants: []grant.Grant{g1},
				StartingSnapshot: balance.Snapshot{
					Balances: balance.Map{g1.ID: 100.0},
					Overage:  0,
					At:       t1,
				},
				Until:  t1.AddDate(0, 0, 1),
				Resets: timeutil.NewSimpleTimeline([]time.Time{t1}),
			},
		)
		assert.EqualError(t, err, "provided reset times must occur after the starting snapshot, got 2024-01-01 00:00:00 +0000 UTC")
	})
}
