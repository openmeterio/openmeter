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
		Slug: meterSlug,
	}

	grant1 := makeGrant(grant.Grant{
		ID:          "grant-1",
		Amount:      100.0,
		Priority:    1,
		EffectiveAt: t1,
		Expiration: grant.ExpirationPeriod{
			Duration: grant.ExpirationPeriodDurationDay,
			Count:    30,
		},
	})

	// grant2 := makeGrant(grant.Grant{
	// 	ID:          "grant-2",
	// 	Amount:      100.0,
	// 	Priority:    1,
	// 	EffectiveAt: t1,
	// 	Expiration: grant.ExpirationPeriod{
	// 		Duration: grant.ExpirationPeriodDurationDay,
	// 		Count:    30,
	// 	},
	// })

	// Tests with single engine
	tt := []struct {
		name string
		run  func(t *testing.T, eng engine.Engine, use addUsageFunc)
	}{
		{
			name: "Should reset roll over grant balance after one reset",
			run: func(t *testing.T, eng engine.Engine, use addUsageFunc) {
				use(10.0, t1.Add(time.Hour))

				g1 := grant1
				g1.ResetMaxRollover = 50.0

				res, err := eng.Run(
					context.Background(),
					engine.RunParams{
						Grants: []grant.Grant{g1},
						StartingSnapshot: balance.Snapshot{
							Balances: balance.Map{
								g1.ID: 100.0,
							},
							Overage: 0,
							At:      t1,
						},
						Until: t1.AddDate(0, 0, 1),
						ResetBehavior: engine.ResetBehavior{
							PreserveOverage: false,
						},
						Resets: timeutil.NewTimeline([]time.Time{t1.Add(time.Hour * 5)}),
					},
				)
				assert.NoError(t, err)

				// The grant should be rolled over:
				// 100 - 10 = 90;
				// Min(50, max(0, 90)) = 50
				assert.Equal(t, 50.0, res.Snapshot.Balances[grant1.ID])
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
			tc.run(t, engine.NewEngine(engine.EngineConfig{
				QueryUsage:  queryFeatureUsage,
				Granularity: meterpkg.WindowSizeMinute,
			}), func(usage float64, at time.Time) {
				streamingConnector.AddSimpleEvent(meterSlug, usage, at)
			})
		})
	}
}
