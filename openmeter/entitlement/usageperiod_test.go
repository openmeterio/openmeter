package entitlement_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestUsagePeriodValidation(t *testing.T) {
	startTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("should be valid", func(t *testing.T) {
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return startTime })(timeutil.Recurrence{
				Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)},
				Anchor:   startTime,
			}),
		})

		require.NoError(t, up.Validate())
	})

	t.Run("should be invalid if no recurrences", func(t *testing.T) {
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{})

		require.Error(t, up.Validate())
	})

	t.Run("should be invalid if recurrence interval is negative", func(t *testing.T) {
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return startTime })(timeutil.Recurrence{
				Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, -1, 0, 0)},
				Anchor:   startTime,
			}),
		})

		require.Error(t, up.Validate())
	})

	t.Run("should be invalid if recurrence anchor is zero", func(t *testing.T) {
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return startTime })(timeutil.Recurrence{
				Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)},
				Anchor:   time.Time{},
			}),
		})

		require.Error(t, up.Validate())
	})
}

func TestUsagePeriodGetPeriodAt(t *testing.T) {
	clock.ResetTime()
	clock.UnFreeze()

	t.Run("should return recurrence.GetPeriodAt for single recurrence value", func(t *testing.T) {
		now := clock.Now()

		// lets fuzz this a bit
		for i := 0; i < 100; i++ {
			someTime := gofakeit.DateRange(now.AddDate(0, -1, 0), now)
			someTime2 := gofakeit.DateRange(now, now.AddDate(0, 1, 0))

			rec := timeutil.Recurrence{
				Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)},
				Anchor:   someTime,
			}

			up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
				timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return someTime })(rec),
			})

			p1, err := rec.GetPeriodAt(someTime2)
			require.NoError(t, err)

			p2, err := up.GetCurrentPeriodAt(someTime2)
			require.NoError(t, err)

			require.Equal(t, p1, p2)
		}
	})

	t.Run("should return the first period if querying before the first recurrence", func(t *testing.T) {
		now := clock.Now()

		t1 := now.AddDate(0, -3, 0)
		t2 := now
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		t1StartOfDay := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, t1.Location())

		rec1 := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   startOfDay.Add(time.Hour),
		}

		rec1FirstPeriodEnd := datetime.NewDateTime(t1StartOfDay).AddDateNoOverflow(0, 1, 0).Time.Add(time.Hour)

		rec2 := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   startOfDay.Add(time.Hour * 2),
		}

		// We register 3 reset times along the past 3 years
		// each with different anchor times (we'll use the hour part to assert the correct recurrence is used)
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t1 })(rec1),
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t2 })(rec2),
		})

		timeInPast := gofakeit.DateRange(
			datetime.NewDateTime(t1).AddDateNoOverflow(-1, 0, 1).Time,
			datetime.NewDateTime(t1).AddDateNoOverflow(0, 0, -1).Time)

		period, err := up.GetCurrentPeriodAt(timeInPast)
		require.NoError(t, err)

		// Should return the first period even when queried for past
		expected := timeutil.ClosedPeriod{
			From: t1,
			To:   rec1FirstPeriodEnd,
		}

		require.Equal(t, expected, period, `
		now: %s
		startOfDay: %s
		t1: %s
		t1StartOfDay: %s
		t2: %s
		timeInPast: %s
		expected: %+v
		got: %+v
		`, now, startOfDay, t1, t1StartOfDay, t2, timeInPast, expected, period)
	})

	t.Run("should find the correct recurrence to use when multiple are present", func(t *testing.T) {
		// lets fuzz this a bit

		for i := 0; i < 300; i++ {
			now := time.Date(2025, 6, 18, 11, 23, 0, 0, time.UTC)
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

			t1 := now.AddDate(-2, 0, 0)
			t2 := now.AddDate(-1, 0, 0)
			t3 := now

			rec2 := timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   startOfDay.Add(time.Hour * 2),
			}

			// We register 3 reset times along the past 3 years
			// each with different anchor times (we'll use the hour part to assert the correct recurrence is used)
			up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
				timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t1 })(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   startOfDay.Add(time.Hour),
				}),
				timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t2 })(rec2),
				timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t3 })(timeutil.Recurrence{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   startOfDay.Add(time.Hour * 3),
				}),
			})

			// Let's make sure we're not falling on a boundary period (as those would be truncated)
			timeInMiddle := gofakeit.DateRange(now.AddDate(-1, 1, 1), now.AddDate(0, -1, -1))

			period, err := up.GetCurrentPeriodAt(timeInMiddle)
			require.NoError(t, err)

			recPeriod, err := rec2.GetPeriodAt(timeInMiddle)
			require.NoError(t, err)

			require.Equal(t, recPeriod, period, "iteration %d, looked for ts %s", i, timeInMiddle)
		}
	})

	t.Run("should truncate the returned period according to the recurrence boundaries", func(t *testing.T) {
		now := clock.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		t1 := now.AddDate(0, 0, -26)
		t2 := now.AddDate(0, 0, -13)
		t3 := now

		rec2 := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   startOfDay.Add(time.Hour * 2),
		}

		// We register 3 reset times along the past 3 years
		// each with different anchor times (we'll use the hour part to assert the correct recurrence is used)
		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t1 })(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   startOfDay.Add(time.Hour),
			}),
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t2 })(rec2),
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return t3 })(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   startOfDay.Add(time.Hour * 3),
			}),
		})

		timeInMiddle := gofakeit.DateRange(now.AddDate(0, 0, -25), now.AddDate(0, 0, -14))

		period, err := up.GetCurrentPeriodAt(timeInMiddle)
		require.NoError(t, err)

		require.Equal(t, timeutil.ClosedPeriod{
			From: t1,
			To:   t2,
		}, period)
	})
}

func TestUsagePeriodGetResetTimelineInclusive(t *testing.T) {
	t.Run("Should return later reset times for single recurrence when period is not aligned with anchor", func(t *testing.T) {
		now := time.Date(2025, 6, 5, 14, 8, 2, 0, time.UTC)
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		anchor := startOfDay.Add(time.Hour)

		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return startOfDay })(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily, // For simplicity we'll use daily
				Anchor:   anchor,
			}),
		})

		t.Run("misaligned period", func(t *testing.T) {
			queryPeriod := timeutil.ClosedPeriod{
				From: now.AddDate(0, 0, 1),
				To:   now.AddDate(0, 0, 5),
			}
			timeline, err := up.GetResetTimelineInclusive(queryPeriod)
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 4, "queried for period %+v", queryPeriod)

			require.Equal(t, anchor.AddDate(0, 0, 2), timeline.GetTimes()[0])
			require.Equal(t, anchor.AddDate(0, 0, 3), timeline.GetTimes()[1])
			require.Equal(t, anchor.AddDate(0, 0, 4), timeline.GetTimes()[2])
			require.Equal(t, anchor.AddDate(0, 0, 5), timeline.GetTimes()[3])
		})

		t.Run("aligned period", func(t *testing.T) {
			queryPeriod := timeutil.ClosedPeriod{
				From: anchor.AddDate(0, 0, 1),
				To:   anchor.AddDate(0, 0, 5),
			}
			timeline, err := up.GetResetTimelineInclusive(queryPeriod)
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 5, "queried for period %+v", queryPeriod)

			require.Equal(t, anchor.AddDate(0, 0, 1), timeline.GetTimes()[0])
			require.Equal(t, anchor.AddDate(0, 0, 2), timeline.GetTimes()[1])
			require.Equal(t, anchor.AddDate(0, 0, 3), timeline.GetTimes()[2])
			require.Equal(t, anchor.AddDate(0, 0, 4), timeline.GetTimes()[3])
			require.Equal(t, anchor.AddDate(0, 0, 5), timeline.GetTimes()[4])
		})
	})

	t.Run("Should not return values before the first recurrence, but should return the first recurrence", func(t *testing.T) {
		now := time.Date(2025, 6, 5, 14, 8, 2, 0, time.UTC)
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return now })(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily, // For simplicity we'll use daily
				Anchor:   startOfDay.Add(time.Hour),
			}),
		})

		t.Run("misaligned", func(t *testing.T) {
			queryPeriod := timeutil.ClosedPeriod{
				From: startOfDay.AddDate(0, -1, 0), // past
				To:   now.AddDate(0, 0, 3),
			}

			timeline, err := up.GetResetTimelineInclusive(queryPeriod)
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 4, "queried for period %+v", queryPeriod)

			require.Equal(t, now, timeline.GetTimes()[0])
		})

		t.Run("exact on first recurrence", func(t *testing.T) {
			queryPeriod := timeutil.ClosedPeriod{
				From: now, // first recurrence
				To:   now.AddDate(0, 0, 3),
			}

			timeline, err := up.GetResetTimelineInclusive(queryPeriod)
			require.NoError(t, err)

			require.Len(t, timeline.GetTimes(), 4, "queried for period %+v", queryPeriod)

			require.Equal(t, now, timeline.GetTimes()[0])
		})
	})

	t.Run("Should handle multiple recurrences", func(t *testing.T) {
		now := time.Date(2025, 6, 5, 14, 8, 2, 0, time.UTC)
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		rec1 := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   startOfDay.Add(time.Hour),
		}

		rec2 := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   startOfDay.Add(time.Hour * 2),
		}

		up := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return now })(rec1),
			timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return now.AddDate(0, 0, 3) })(rec2),
		})

		queryPeriod := timeutil.ClosedPeriod{
			From: startOfDay,
			To:   startOfDay.AddDate(0, 0, 5),
		}

		timeline, err := up.GetResetTimelineInclusive(queryPeriod)
		require.NoError(t, err)

		require.Len(t, timeline.GetTimes(), 6, "queried for period %+v", queryPeriod)

		require.Equal(t, now, timeline.GetTimes()[0])
		require.Equal(t, startOfDay.AddDate(0, 0, 1).Add(time.Hour), timeline.GetTimes()[1])
		require.Equal(t, startOfDay.AddDate(0, 0, 2).Add(time.Hour), timeline.GetTimes()[2])
		require.Equal(t, startOfDay.AddDate(0, 0, 3).Add(time.Hour), timeline.GetTimes()[3])
		require.Equal(t, now.AddDate(0, 0, 3), timeline.GetTimes()[4])
		require.Equal(t, startOfDay.AddDate(0, 0, 4).Add(time.Hour*2), timeline.GetTimes()[5])
	})
}

func TestUsagePeriodSerialization(t *testing.T) {
	complexUsagePeriod := entitlement.NewUsagePeriod([]timeutil.Timed[timeutil.Recurrence]{
		timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return r.Anchor })(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily, // For simplicity we'll use daily
			Anchor:   time.Date(2025, 6, 18, 11, 23, 0, 0, time.UTC),
		}),
		timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return r.Anchor })(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily, // For simplicity we'll use daily
			Anchor:   time.Date(2025, 8, 18, 10, 23, 0, 0, time.UTC),
		}),
		timeutil.AsTimed(func(r timeutil.Recurrence) time.Time { return r.Anchor })(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily, // For simplicity we'll use daily
			Anchor:   time.Date(2025, 10, 18, 10, 23, 0, 0, time.UTC),
		}),
	})

	serialized, err := json.Marshal(complexUsagePeriod)
	require.NoError(t, err)

	var deserialized entitlement.UsagePeriod
	err = json.Unmarshal(serialized, &deserialized)
	require.NoError(t, err)

	require.True(t, deserialized.Equal(complexUsagePeriod), "\ndeserialized: %+v, \n\noriginal: %+v\n", deserialized, complexUsagePeriod)
}
