package planhistory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription/planhistory"
)

func TestHistoryAt(t *testing.T) {
	// given: recorded changes, including a scheduled version, in nonchronological order.
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	boundary := start.Add(time.Hour)
	history := planhistory.History{
		{EffectiveAt: boundary, Plan: planhistory.PlanVersion{Key: "pro", Version: 2}},
		{EffectiveAt: start, Plan: planhistory.PlanVersion{Key: "pro", Version: 1}},
	}

	// when: callers resolve attribution on either side of the effective boundary.
	before := history.At(boundary.Add(-time.Nanosecond))
	after := history.At(boundary)

	// then: future versions cannot rewrite earlier service, and unknown history stays unknown.
	require.Equal(t, &planhistory.PlanVersion{Key: "pro", Version: 1}, before)
	require.Equal(t, &planhistory.PlanVersion{Key: "pro", Version: 2}, after)
	require.Nil(t, history.At(start.Add(-time.Nanosecond)))
	require.Nil(t, planhistory.History(nil).At(start))
	history[0].Plan.Version = 99
	require.Equal(t, 2, after.Version)
}
