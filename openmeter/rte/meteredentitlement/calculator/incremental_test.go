package calculator

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestOnePhaseNeedsOnlyItsOrdinaryQuery(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	phase := timeutil.ClosedPeriod{From: start, To: start.Add(2 * time.Hour)}
	history := emptyUsageWindows{
		until: phase.To, candidates: []timeutil.ClosedPeriod{{From: start, To: start.Add(time.Hour)}},
	}

	calls := 0
	value, err := history.usage(phase, balance.PhaseUsageReader{
		Value: func(p timeutil.ClosedPeriod) (float64, bool, error) {
			calls++
			require.Equal(t, phase, p)

			return 9, false, nil
		},
		HasEvents: func(timeutil.ClosedPeriod) (bool, error) {
			t.Fatal("a single phase spanning the hint needs no source probe")
			return false, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, 9.0, value)
	require.Equal(t, 1, calls)
	require.False(t, history.reused)
}

func TestFailedEmptinessCheckFallsBackOnce(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	history := emptyUsageWindows{until: end, candidates: []timeutil.ClosedPeriod{{From: start, To: end}}}

	readCalls, probes := 0, 0
	reader := balance.PhaseUsageReader{
		Value: func(timeutil.ClosedPeriod) (float64, bool, error) {
			readCalls++
			return 3, false, nil
		},
		HasEvents: func(timeutil.ClosedPeriod) (bool, error) {
			probes++
			return false, errors.New("raw data unavailable")
		},
	}
	for from := start; from.Before(end); from = from.Add(time.Hour) {
		value, err := history.usage(timeutil.ClosedPeriod{From: from, To: from.Add(time.Hour)}, reader)
		require.NoError(t, err)
		require.Equal(t, 3.0, value)
	}

	require.False(t, history.reused)
	require.Equal(t, 2, readCalls)
	require.Equal(t, 1, probes)
}

func TestUnsupportedRowCheckDropsHints(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	history := emptyUsageWindows{until: end, candidates: []timeutil.ClosedPeriod{
		{From: start, To: end},
		{From: start, To: end.Add(-time.Hour)},
	}}

	reader := balance.PhaseUsageReader{
		Value:     func(timeutil.ClosedPeriod) (float64, bool, error) { return 0, true, nil },
		HasEvents: func(timeutil.ClosedPeriod) (bool, error) { return false, errors.ErrUnsupported },
	}
	for from := start; from.Before(end); from = from.Add(time.Hour) {
		_, err := history.usage(timeutil.ClosedPeriod{From: from, To: from.Add(time.Hour)}, reader)
		require.NoError(t, err)
	}

	require.ElementsMatch(t, []time.Time{end, end.Add(-time.Hour)}, history.discarded)
	require.Zero(t, history.best)
	require.False(t, history.reused)
}
