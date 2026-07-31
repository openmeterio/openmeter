package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGrantBurnDownHistory_ChunkByResets(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	usageAtStart := balance.SnapshottedUsage{
		Since: start.Add(-time.Hour),
		Usage: 7,
	}

	segment := func(idx int, grantUsage float64, reset bool) engine.GrantBurnDownHistorySegment {
		from := start.Add(time.Duration(idx) * time.Hour)

		return engine.GrantBurnDownHistorySegment{
			ClosedPeriod: timeutil.ClosedPeriod{
				From: from,
				To:   from.Add(time.Hour),
			},
			TotalUsage: grantUsage + 1,
			GrantUsages: []engine.GrantUsage{
				{
					GrantID: "grant-1",
					Usage:   grantUsage,
				},
			},
			TerminationReasons: engine.SegmentTerminationReason{
				UsageReset: reset,
			},
		}
	}

	t.Run("Empty history returns no chunks", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory(nil, usageAtStart)
		require.NoError(t, err)

		assert.Empty(t, history.ChunkByResets())
	})

	t.Run("History without resets stays as one chunk", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, false),
			segment(1, 20, false),
		}, usageAtStart)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 1)
		assert.Len(t, chunks[0].Segments(), 2)
		assert.Equal(t, 30.0, chunks[0].TotalGrantUsage().InexactFloat64())

		usage, err := chunks[0].GetUsageInPeriodUntilSegment(0)
		require.NoError(t, err)
		assert.Equal(t, usageAtStart, usage)
	})

	t.Run("History is chunked after reset segments", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, false),
			segment(1, 20, true),
			segment(2, 30, false),
			segment(3, 40, true),
			segment(4, 50, false),
		}, usageAtStart)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 3)

		assert.Len(t, chunks[0].Segments(), 2)
		assert.Equal(t, 30.0, chunks[0].TotalGrantUsage().InexactFloat64())
		assertChunkUsageAtStart(t, chunks[0], usageAtStart)

		assert.Len(t, chunks[1].Segments(), 2)
		assert.Equal(t, 70.0, chunks[1].TotalGrantUsage().InexactFloat64())
		assertChunkUsageAtStart(t, chunks[1], balance.SnapshottedUsage{
			Since: start.Add(2 * time.Hour),
			Usage: 0,
		})

		assert.Len(t, chunks[2].Segments(), 1)
		assert.Equal(t, 50.0, chunks[2].TotalGrantUsage().InexactFloat64())
		assertChunkUsageAtStart(t, chunks[2], balance.SnapshottedUsage{
			Since: start.Add(4 * time.Hour),
			Usage: 0,
		})
	})

	t.Run("Final reset does not create empty trailing chunk", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, true),
		}, usageAtStart)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 1)
		assert.Len(t, chunks[0].Segments(), 1)
		assert.Equal(t, 10.0, chunks[0].TotalGrantUsage().InexactFloat64())
	})

	t.Run("Rollover starts the new usage period chunk", func(t *testing.T) {
		resetAt := start.Add(time.Hour)
		beforeReset := segment(0, 10, true)
		rollover := engine.GrantBurnDownHistorySegment{
			ClosedPeriod: timeutil.ClosedPeriod{
				From: resetAt,
				To:   resetAt,
			},
			TerminationReasons: engine.SegmentTerminationReason{
				Rollover: true,
			},
			GrantUsages: []engine.GrantUsage{
				{
					GrantID: "grant-1",
					Usage:   5,
				},
			},
		}
		afterReset := segment(1, 20, false)

		history, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{beforeReset, rollover, afterReset},
			usageAtStart,
		)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 2)
		require.Len(t, chunks[1].Segments(), 2)
		assert.True(t, chunks[1].Segments()[0].TerminationReasons.Rollover)
		assert.Equal(t, 25.0, chunks[1].TotalGrantUsage().InexactFloat64())
		assertChunkUsageAtStart(t, chunks[1], balance.SnapshottedUsage{
			Since: resetAt,
			Usage: 0,
		})
	})
}

func TestGrantBurnDownHistory_RolloverPrecedesUsageAtSameTime(t *testing.T) {
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	usageSegment := engine.GrantBurnDownHistorySegment{
		ClosedPeriod: timeutil.ClosedPeriod{
			From: at,
			To:   at.Add(time.Hour),
		},
	}
	rolloverSegment := engine.GrantBurnDownHistorySegment{
		ClosedPeriod: timeutil.ClosedPeriod{
			From: at,
			To:   at,
		},
		TerminationReasons: engine.SegmentTerminationReason{
			Rollover: true,
		},
	}

	history, err := engine.NewGrantBurnDownHistory(
		[]engine.GrantBurnDownHistorySegment{usageSegment, rolloverSegment},
		balance.SnapshottedUsage{},
	)
	require.NoError(t, err)
	require.Len(t, history.Segments(), 2)
	assert.True(t, history.Segments()[0].TerminationReasons.Rollover)
	assert.False(t, history.Segments()[1].TerminationReasons.Rollover)
}

func assertChunkUsageAtStart(t *testing.T, history engine.GrantBurnDownHistory, expected balance.SnapshottedUsage) {
	t.Helper()

	usage, err := history.GetUsageInPeriodUntilSegment(0)
	require.NoError(t, err)
	assert.Equal(t, expected, usage)
}
