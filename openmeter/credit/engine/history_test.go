package engine_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/unitconfig"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGrantBurnDownHistory_ChunkByResets(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	usageAtStart := balance.SnapshottedUsage{
		Since: start.Add(-time.Hour),
		Usage: 7,
	}
	startingUnitConfig := &unitconfig.UnitConfig{
		Operation:        unitconfig.UnitConfigOperationMultiply,
		ConversionFactor: alpacadecimal.NewFromInt(2),
	}
	startingSnapshot := balance.Snapshot{
		At:    start,
		Usage: usageAtStart,
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           usageAtStart.Usage,
			TotalGrantUsage: 6,
		},
		UnitConfig: startingUnitConfig,
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
		history, err := engine.NewGrantBurnDownHistory(nil, startingSnapshot)
		require.NoError(t, err)

		assert.Empty(t, history.ChunkByResets())
	})

	t.Run("History without resets stays as one chunk", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, false),
			segment(1, 20, false),
		}, startingSnapshot)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 1)
		assert.Len(t, chunks[0].Segments(), 2)
		assert.Equal(t, 30.0, chunks[0].TotalGrantUsage().InexactFloat64())

		assertChunkSnapshotAtStart(t, chunks[0], usageAtStart, 6, startingUnitConfig)
	})

	t.Run("History is chunked after reset segments", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, false),
			segment(1, 20, true),
			segment(2, 30, false),
			segment(3, 40, true),
			segment(4, 50, false),
		}, startingSnapshot)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 3)

		assert.Len(t, chunks[0].Segments(), 2)
		assert.Equal(t, 30.0, chunks[0].TotalGrantUsage().InexactFloat64())
		assertChunkSnapshotAtStart(t, chunks[0], usageAtStart, 6, startingUnitConfig)

		assert.Len(t, chunks[1].Segments(), 2)
		assert.Equal(t, 70.0, chunks[1].TotalGrantUsage().InexactFloat64())
		assertChunkSnapshotAtStart(t, chunks[1], balance.SnapshottedUsage{
			Since: start.Add(2 * time.Hour),
			Usage: 0,
		}, 0, startingUnitConfig)

		assert.Len(t, chunks[2].Segments(), 1)
		assert.Equal(t, 50.0, chunks[2].TotalGrantUsage().InexactFloat64())
		assertChunkSnapshotAtStart(t, chunks[2], balance.SnapshottedUsage{
			Since: start.Add(4 * time.Hour),
			Usage: 0,
		}, 0, startingUnitConfig)
	})

	t.Run("Final reset does not create empty trailing chunk", func(t *testing.T) {
		history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
			segment(0, 10, true),
		}, startingSnapshot)
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
			startingSnapshot,
		)
		require.NoError(t, err)

		chunks := history.ChunkByResets()
		require.Len(t, chunks, 2)
		require.Len(t, chunks[1].Segments(), 2)
		assert.True(t, chunks[1].Segments()[0].TerminationReasons.Rollover)
		assert.Equal(t, 25.0, chunks[1].TotalGrantUsage().InexactFloat64())
		assertChunkSnapshotAtStart(t, chunks[1], balance.SnapshottedUsage{
			Since: resetAt,
			Usage: 0,
		}, 0, startingUnitConfig)

		snapshotAfterRollover, err := history.GetSnapshotAtStartOfSegment(2)
		require.NoError(t, err)
		require.NotNil(t, snapshotAfterRollover.UsageSnapshot)
		assert.Equal(t, balance.UsageSnapshot{
			TotalGrantUsage: 5,
		}, *snapshotAfterRollover.UsageSnapshot)
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
		balance.Snapshot{At: at, UsageSnapshot: zeroUsageSnapshot()},
	)
	require.NoError(t, err)
	require.Len(t, history.Segments(), 2)
	assert.True(t, history.Segments()[0].TerminationReasons.Rollover)
	assert.False(t, history.Segments()[1].TerminationReasons.Rollover)
}

func TestNewGrantBurnDownHistory_ValidatesAnchorAndContinuity(t *testing.T) {
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	startingSnapshot := balance.Snapshot{
		At:            at,
		Balances:      balance.Map{"grant-1": 100},
		Overage:       5,
		UsageSnapshot: zeroUsageSnapshot(),
	}
	first := engine.GrantBurnDownHistorySegment{
		ClosedPeriod: timeutil.ClosedPeriod{
			From: at,
			To:   at.Add(time.Hour),
		},
		// Boundary changes such as recurrence can make this differ from the
		// starting snapshot without breaking the history anchor.
		BalanceAtStart: balance.Map{"grant-1": 200},
		OverageAtStart: 5,
	}
	second := engine.GrantBurnDownHistorySegment{
		ClosedPeriod: timeutil.ClosedPeriod{
			From: at.Add(time.Hour),
			To:   at.Add(2 * time.Hour),
		},
	}

	t.Run("aligned continuous history", func(t *testing.T) {
		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{second, first},
			startingSnapshot,
		)
		require.NoError(t, err)
	})

	t.Run("empty history", func(t *testing.T) {
		_, err := engine.NewGrantBurnDownHistory(nil, startingSnapshot)
		require.NoError(t, err)
	})

	t.Run("missing usage snapshot", func(t *testing.T) {
		invalid := startingSnapshot
		invalid.UsageSnapshot = nil

		_, err := engine.NewGrantBurnDownHistory(nil, invalid)
		require.ErrorContains(t, err, "usage snapshot is missing")
	})

	t.Run("first segment does not start at snapshot", func(t *testing.T) {
		misaligned := first
		misaligned.ClosedPeriod = timeutil.ClosedPeriod{
			From: at.Add(time.Minute),
			To:   first.To,
		}

		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{misaligned},
			startingSnapshot,
		)
		require.ErrorContains(t, err, "expected starting snapshot")
	})

	t.Run("first segment overage does not match snapshot", func(t *testing.T) {
		misaligned := first
		misaligned.OverageAtStart = 6

		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{misaligned},
			startingSnapshot,
		)
		require.ErrorContains(t, err, "expected starting snapshot overage")
	})

	t.Run("segment starts after it ends", func(t *testing.T) {
		invalid := first
		invalid.ClosedPeriod = timeutil.ClosedPeriod{
			From: at,
			To:   at.Add(-time.Minute),
		}

		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{invalid},
			startingSnapshot,
		)
		require.ErrorContains(t, err, "starts after it ends")
	})

	t.Run("segments have a gap", func(t *testing.T) {
		gapped := second
		gapped.ClosedPeriod = timeutil.ClosedPeriod{
			From: second.From.Add(time.Minute),
			To:   second.To,
		}

		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{first, gapped},
			startingSnapshot,
		)
		require.ErrorContains(t, err, "are not contiguous")
	})

	t.Run("segments overlap", func(t *testing.T) {
		overlapping := second
		overlapping.ClosedPeriod = timeutil.ClosedPeriod{
			From: second.From.Add(-time.Minute),
			To:   second.To,
		}

		_, err := engine.NewGrantBurnDownHistory(
			[]engine.GrantBurnDownHistorySegment{first, overlapping},
			startingSnapshot,
		)
		require.ErrorContains(t, err, "are not contiguous")
	})
}

func assertChunkSnapshotAtStart(
	t *testing.T,
	history engine.GrantBurnDownHistory,
	expectedUsage balance.SnapshottedUsage,
	expectedTotalGrantUsage float64,
	expectedUnitConfig *unitconfig.UnitConfig,
) {
	t.Helper()

	snapshot, err := history.GetSnapshotAtStartOfSegment(0)
	require.NoError(t, err)
	assert.Equal(t, expectedUsage, snapshot.Usage)
	require.NotNil(t, snapshot.UsageSnapshot)
	assert.Equal(t, balance.UsageSnapshot{
		Usage:           expectedUsage.Usage,
		TotalGrantUsage: expectedTotalGrantUsage,
	}, *snapshot.UsageSnapshot)
	assert.True(t, expectedUnitConfig.Equal(snapshot.UnitConfig))
}
