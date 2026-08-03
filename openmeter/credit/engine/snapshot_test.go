package engine_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func zeroUsageSnapshot() *balance.UsageSnapshot {
	return lo.ToPtr(balance.UsageSnapshot{})
}

func TestEngineValidatesStartingUsageSnapshot(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		usageSnapshot *balance.UsageSnapshot
		valid         bool
	}{
		{
			name: "missing",
		},
		{
			name: "negative total grant usage",
			usageSnapshot: &balance.UsageSnapshot{
				TotalGrantUsage: -1,
			},
		},
		{
			name: "total grant usage is not a number",
			usageSnapshot: &balance.UsageSnapshot{
				TotalGrantUsage: math.NaN(),
			},
		},
		{
			name: "total grant usage is infinite",
			usageSnapshot: &balance.UsageSnapshot{
				TotalGrantUsage: math.Inf(1),
			},
		},
		{
			name:          "explicit zero",
			usageSnapshot: &balance.UsageSnapshot{},
			valid:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usageQueried := false
			eng := engine.NewEngine(engine.EngineConfig{
				QueryUsage: func(_ context.Context, _, _ time.Time) (float64, error) {
					usageQueried = true
					return 0, nil
				},
			})

			_, err := eng.Run(t.Context(), engine.RunParams{
				StartingSnapshot: balance.Snapshot{
					At:            start,
					Balances:      balance.Map{},
					UsageSnapshot: tt.usageSnapshot,
				},
				Until: start.Add(time.Hour),
			})

			if tt.valid {
				require.NoError(t, err)
				assert.True(t, usageQueried)
				return
			}

			require.ErrorContains(t, err, "starting snapshot")
			assert.False(t, usageQueried)
		})
	}
}

func TestEngineAccumulatesTotalGrantUsageFromSnapshot(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	startingTotalGrantUsage := 25.0
	g := grant.Grant{
		ID:          "grant-1",
		Amount:      100,
		Priority:    1,
		EffectiveAt: start,
	}
	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(_ context.Context, _, _ time.Time) (float64, error) {
			return 10, nil
		},
	})

	result, err := eng.Run(t.Context(), engine.RunParams{
		Meter: meter.Meter{
			Aggregation: meter.MeterAggregationSum,
		},
		Grants: []grant.Grant{g},
		StartingSnapshot: balance.Snapshot{
			At:       start,
			Balances: balance.Map{g.ID: g.Amount},
			UsageSnapshot: &balance.UsageSnapshot{
				TotalGrantUsage: startingTotalGrantUsage,
			},
		},
		Until: start.Add(time.Hour),
	})
	require.NoError(t, err)
	require.NotNil(t, result.Snapshot.UsageSnapshot)
	assert.Equal(t, 10.0, result.Snapshot.UsageSnapshot.Usage)
	assert.Equal(t, 35.0, result.Snapshot.UsageSnapshot.TotalGrantUsage)

	historyStart, err := result.History.GetSnapshotAtStartOfSegment(0)
	require.NoError(t, err)
	require.NotNil(t, historyStart.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		TotalGrantUsage: 25,
	}, *historyStart.UsageSnapshot)
}

func TestEngineResetsTotalGrantUsageBeforeRolloverOverageBurn(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	resetAt := start.Add(time.Hour)
	end := resetAt.Add(time.Hour)
	startingTotalGrantUsage := 25.0
	g := grant.Grant{
		ID:               "grant-1",
		Amount:           100,
		Priority:         1,
		EffectiveAt:      start,
		ResetMinRollover: 50,
		ResetMaxRollover: 50,
	}
	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(_ context.Context, from, _ time.Time) (float64, error) {
			if from.Before(resetAt) {
				return 110, nil
			}

			return 5, nil
		},
	})

	result, err := eng.Run(t.Context(), engine.RunParams{
		Meter: meter.Meter{
			Aggregation: meter.MeterAggregationSum,
		},
		Grants: []grant.Grant{g},
		StartingSnapshot: balance.Snapshot{
			At:       start,
			Balances: balance.Map{g.ID: g.Amount},
			UsageSnapshot: &balance.UsageSnapshot{
				TotalGrantUsage: startingTotalGrantUsage,
			},
		},
		Until: end,
		ResetBehavior: grant.ResetBehavior{
			PreserveOverage: true,
		},
		Resets: timeutil.NewSimpleTimeline([]time.Time{resetAt}),
	})
	require.NoError(t, err)
	require.Len(t, result.History.Segments(), 3)

	rolloverStart, err := result.History.GetSnapshotAtStartOfSegment(1)
	require.NoError(t, err)
	require.NotNil(t, rolloverStart.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{}, *rolloverStart.UsageSnapshot)

	afterRollover, err := result.History.GetSnapshotAtStartOfSegment(2)
	require.NoError(t, err)
	require.NotNil(t, afterRollover.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		TotalGrantUsage: 10,
	}, *afterRollover.UsageSnapshot)

	require.NotNil(t, result.Snapshot.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           5,
		TotalGrantUsage: 15,
	}, *result.Snapshot.UsageSnapshot)
}

func TestEnginePreservesUsageSnapshotAcrossRuns(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	intermediate := start.Add(time.Hour)
	end := intermediate.Add(time.Hour)
	g := grant.Grant{
		ID:          "grant-1",
		Amount:      100,
		Priority:    1,
		EffectiveAt: start,
	}
	startingSnapshot := balance.Snapshot{
		At:       start,
		Balances: balance.Map{g.ID: 95},
		Usage: balance.SnapshottedUsage{
			Since: start,
			Usage: 5,
		},
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           5,
			TotalGrantUsage: 5,
		},
	}
	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(_ context.Context, from, to time.Time) (float64, error) {
			return to.Sub(from).Hours() * 10, nil
		},
	})
	runParams := engine.RunParams{
		Meter: meter.Meter{
			Aggregation: meter.MeterAggregationSum,
		},
		Grants:           []grant.Grant{g},
		StartingSnapshot: startingSnapshot,
		Until:            end,
	}

	singleRun, err := eng.Run(t.Context(), runParams)
	require.NoError(t, err)

	firstRunParams := runParams
	firstRunParams.Until = intermediate
	firstRun, err := eng.Run(t.Context(), firstRunParams)
	require.NoError(t, err)

	secondRunParams := runParams
	secondRunParams.StartingSnapshot = firstRun.Snapshot
	secondRun, err := eng.Run(t.Context(), secondRunParams)
	require.NoError(t, err)

	require.NotNil(t, singleRun.Snapshot.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           25,
		TotalGrantUsage: 25,
	}, *singleRun.Snapshot.UsageSnapshot)
	assert.Equal(t, singleRun.Snapshot, secondRun.Snapshot)
}

func TestEnginePreservesUsageSnapshotWhenResumingAfterRollover(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	resetAt := start.Add(time.Hour)
	end := resetAt.Add(time.Hour)
	g := grant.Grant{
		ID:               "grant-1",
		Amount:           100,
		Priority:         1,
		EffectiveAt:      start,
		ResetMinRollover: 50,
		ResetMaxRollover: 50,
	}
	eng := engine.NewEngine(engine.EngineConfig{
		QueryUsage: func(_ context.Context, from, _ time.Time) (float64, error) {
			if from.Before(resetAt) {
				return 110, nil
			}

			return 5, nil
		},
	})
	runParams := engine.RunParams{
		Meter: meter.Meter{
			Aggregation: meter.MeterAggregationSum,
		},
		Grants:           []grant.Grant{g},
		StartingSnapshot: balance.NewStartingSnapshot([]grant.Grant{g}, start),
		Until:            end,
		ResetBehavior: grant.ResetBehavior{
			PreserveOverage: true,
		},
		Resets: timeutil.NewSimpleTimeline([]time.Time{resetAt}),
	}

	singleRun, err := eng.Run(t.Context(), runParams)
	require.NoError(t, err)
	require.Len(t, singleRun.History.Segments(), 3)

	afterRollover, err := singleRun.History.GetSnapshotAtStartOfSegment(2)
	require.NoError(t, err)

	resumedRunParams := runParams
	resumedRunParams.StartingSnapshot = afterRollover
	resumedRunParams.Resets = timeutil.SimpleTimeline{}
	resumedRun, err := eng.Run(t.Context(), resumedRunParams)
	require.NoError(t, err)

	require.NotNil(t, resumedRun.Snapshot.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           5,
		TotalGrantUsage: 15,
	}, *resumedRun.Snapshot.UsageSnapshot)
	assert.Equal(t, singleRun.Snapshot, resumedRun.Snapshot)
}
