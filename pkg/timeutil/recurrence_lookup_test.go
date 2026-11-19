package timeutil_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// The legacy* helpers below reimplement the former linear walk from the anchor. They
// exist only to validate that the estimate-and-correct lookup returns the same
// boundaries as before. Once that implementation has proven itself in production,
// remove these helpers and the legacy comparisons (TestRecurrenceLookupMatchesLinearWalk
// and the linear-walk cases of TestRecurrenceLookupFarFromAnchor), keeping explicit
// expected values for the cases worth retaining.

// legacyBoundary is boundary i of the recurrence: Anchor + i*Interval.
func legacyBoundary(t *testing.T, r timeutil.Recurrence, i int) time.Time {
	t.Helper()

	if i == 0 {
		return r.Anchor
	}

	interval, err := r.Interval.Mul(i)
	require.NoError(t, err)

	at, ok := interval.AddTo(r.Anchor)
	require.True(t, ok)

	return at
}

// legacyNextAfterInclusive walks one interval at a time from the anchor, like the
// original implementation, and returns the first boundary at or after at.
func legacyNextAfterInclusive(t *testing.T, r timeutil.Recurrence, at time.Time) time.Time {
	t.Helper()

	res := r.Anchor

	if r.Anchor.After(at) {
		for i := -1; ; i-- {
			v := legacyBoundary(t, r, i)
			if v.Before(at) {
				return res
			}

			res = v
		}
	}

	for i := 1; res.Before(at); i++ {
		res = legacyBoundary(t, r, i)
	}

	return res
}

// legacyPrevBeforeInclusive walks one interval at a time from the anchor, like the
// original implementation, and returns the last boundary at or before at.
func legacyPrevBeforeInclusive(t *testing.T, r timeutil.Recurrence, at time.Time) time.Time {
	t.Helper()

	res := r.Anchor

	if r.Anchor.After(at) {
		for i := -1; res.After(at); i-- {
			res = legacyBoundary(t, r, i)
		}

		return res
	}

	for i := 1; ; i++ {
		v := legacyBoundary(t, r, i)
		if v.After(at) {
			return res
		}

		res = v
	}
}

type legacyResults struct {
	nextInclusive time.Time
	nextExclusive time.Time
	prevInclusive time.Time
	prevExclusive time.Time
	period        timeutil.ClosedPeriod
}

func legacyLookup(t *testing.T, r timeutil.Recurrence, at time.Time) legacyResults {
	t.Helper()

	res := legacyResults{
		nextInclusive: legacyNextAfterInclusive(t, r, at),
		prevInclusive: legacyPrevBeforeInclusive(t, r, at),
	}

	res.nextExclusive = res.nextInclusive
	if res.nextInclusive.Equal(at) {
		res.nextExclusive = legacyNextAfterInclusive(t, r, at.Add(time.Nanosecond))
	}

	res.prevExclusive = res.prevInclusive
	if res.prevInclusive.Equal(at) {
		res.prevExclusive = legacyPrevBeforeInclusive(t, r, at.Add(-time.Nanosecond))
	}

	res.period = timeutil.ClosedPeriod{From: res.prevInclusive, To: res.nextExclusive}

	return res
}

func assertMatchesLegacy(t *testing.T, r timeutil.Recurrence, at time.Time) {
	t.Helper()

	want := legacyLookup(t, r, at)

	msg := fmt.Sprintf("interval=%s anchor=%s at=%s", r.Interval.ISOString(), r.Anchor.Format(time.RFC3339Nano), at.Format(time.RFC3339Nano))

	nextInclusive, err := r.IterateFromNextAfter(at, timeutil.Inclusive)
	require.NoError(t, err, msg)
	require.True(t, want.nextInclusive.Equal(nextInclusive.At), "next inclusive: want %s, got %s (%s)", want.nextInclusive, nextInclusive.At, msg)

	nextExclusive, err := r.IterateFromNextAfter(at, timeutil.Exclusive)
	require.NoError(t, err, msg)
	require.True(t, want.nextExclusive.Equal(nextExclusive.At), "next exclusive: want %s, got %s (%s)", want.nextExclusive, nextExclusive.At, msg)

	prevInclusive, err := r.IterateFromPrevBefore(at, timeutil.Inclusive)
	require.NoError(t, err, msg)
	require.True(t, want.prevInclusive.Equal(prevInclusive.At), "prev inclusive: want %s, got %s (%s)", want.prevInclusive, prevInclusive.At, msg)

	prevExclusive, err := r.IterateFromPrevBefore(at, timeutil.Exclusive)
	require.NoError(t, err, msg)
	require.True(t, want.prevExclusive.Equal(prevExclusive.At), "prev exclusive: want %s, got %s (%s)", want.prevExclusive, prevExclusive.At, msg)

	period, err := r.GetPeriodAt(at)
	require.NoError(t, err, msg)
	require.True(t, want.period.From.Equal(period.From) && want.period.To.Equal(period.To), "period: want %+v, got %+v (%s)", want.period, period, msg)

	// The returned iterators must stay aligned with their timestamps, so stepping them
	// lands on the neighboring boundaries.
	for _, it := range []timeutil.RecurrenceIterator{nextInclusive, nextExclusive, prevInclusive, prevExclusive} {
		next, err := it.Next()
		require.NoError(t, err, msg)
		require.True(t, legacyNextAfterInclusive(t, r, it.At.Add(time.Nanosecond)).Equal(next.At), "iterator Next from %s: got %s (%s)", it.At, next.At, msg)

		prev, err := it.Prev()
		require.NoError(t, err, msg)
		require.True(t, legacyPrevBeforeInclusive(t, r, it.At.Add(-time.Nanosecond)).Equal(prev.At), "iterator Prev from %s: got %s (%s)", it.At, prev.At, msg)
	}
}

func TestRecurrenceLookupMatchesLinearWalk(t *testing.T) {
	locations := []*time.Location{time.UTC}
	for _, name := range []string{"America/New_York", "Europe/Budapest", "Australia/Lord_Howe"} {
		loc, err := time.LoadLocation(name)
		require.NoError(t, err)

		locations = append(locations, loc)
	}

	intervals := []datetime.ISODuration{
		datetime.NewISODuration(0, 0, 0, 0, 0, 30, 0),
		datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0),
		datetime.NewISODuration(0, 0, 0, 0, 1, 30, 0),
		datetime.NewISODuration(0, 0, 0, 0, 7, 0, 0),
		datetime.DurationDay,
		datetime.NewISODuration(0, 0, 0, 3, 0, 0, 0),
		datetime.DurationWeek,
		datetime.NewISODuration(0, 0, 2, 3, 0, 0, 0),
		datetime.DurationMonth,
		datetime.NewISODuration(0, 1, 0, 15, 0, 0, 0),
		datetime.NewISODuration(0, 3, 0, 0, 0, 0, 0),
		datetime.DurationYear,
		datetime.NewISODuration(1, 6, 0, 0, 0, 0, 0),
	}

	// Month-end and leap-day anchors exercise the no-overflow clamping; the March and
	// November anchors sit on DST transitions in the non-UTC locations.
	//
	// Anchors in the second occurrence of a repeated DST hour are left out on purpose:
	// datetime.DateTime.Add rebuilds the wall clock time, resolving such a time to its
	// first occurrence, so boundaries next to the anchor can collapse onto it.
	fixedAnchors := []string{
		"2024-01-31T15:00:00Z",
		"2024-02-29T00:00:00Z",
		"2023-03-31T23:30:00Z",
		"2024-08-31T12:00:00.123456789Z",
		"2025-03-30T01:30:00Z",
		"2025-11-02T05:30:00Z",
	}

	rng := rand.New(rand.NewPCG(1, 2))
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, interval := range intervals {
		step := interval.DurationApprox()

		for _, loc := range locations {
			anchors := make([]time.Time, 0, len(fixedAnchors)+4)
			for _, s := range fixedAnchors {
				a, err := time.Parse(time.RFC3339Nano, s)
				require.NoError(t, err)

				anchors = append(anchors, a.In(loc))
			}

			for range 4 {
				offset := time.Duration(rng.Int64N(int64(7 * 365 * 24 * time.Hour)))
				anchors = append(anchors, base.Add(offset).Truncate(time.Second).In(loc))
			}

			for _, anchor := range anchors {
				r := timeutil.Recurrence{
					Interval: timeutil.RecurrenceInterval{ISODuration: interval},
					Anchor:   anchor,
				}

				t.Run(fmt.Sprintf("%s/%s/%s", interval.ISOString(), loc, anchor.Format(time.RFC3339Nano)), func(t *testing.T) {
					// Arbitrary times up to 60 intervals before or after the anchor. The
					// legacy walk is linear, so larger distances are covered separately.
					for range 8 {
						offset := time.Duration(rng.Int64N(int64(120*step))) - 60*step
						assertMatchesLegacy(t, r, anchor.Add(offset))
					}

					// Exact boundaries, where inclusive and exclusive lookups differ.
					for range 4 {
						at := legacyBoundary(t, r, rng.IntN(120)-60)
						assertMatchesLegacy(t, r, at)
						assertMatchesLegacy(t, r, at.Add(-time.Nanosecond))
						assertMatchesLegacy(t, r, at.Add(time.Nanosecond))
					}

					assertMatchesLegacy(t, r, anchor)
				})
			}
		}
	}
}

func TestRecurrenceLookupFarFromAnchor(t *testing.T) {
	t.Run("matches the linear walk over a hundred thousand intervals away", func(t *testing.T) {
		r := timeutil.Recurrence{
			Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)},
			Anchor:   time.Date(2014, 1, 31, 10, 0, 0, 0, time.UTC),
		}

		assertMatchesLegacy(t, r, time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC))
		assertMatchesLegacy(t, r, time.Date(2026, 11, 19, 12, 34, 56, 0, time.UTC))
		assertMatchesLegacy(t, r, time.Date(2001, 6, 1, 12, 34, 56, 0, time.UTC))
	})

	t.Run("works beyond the former iteration limit", func(t *testing.T) {
		r := timeutil.Recurrence{
			Interval: timeutil.RecurrenceInterval{ISODuration: datetime.NewISODuration(0, 0, 0, 0, 0, 1, 0)},
			Anchor:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		// About 3.6 million one-minute intervals after the anchor; the linear walk stopped at 1 million.
		at := time.Date(2026, 11, 19, 12, 34, 56, 0, time.UTC)

		period, err := r.GetPeriodAt(at)
		require.NoError(t, err)
		require.Equal(t, timeutil.ClosedPeriod{
			From: time.Date(2026, 11, 19, 12, 34, 0, 0, time.UTC),
			To:   time.Date(2026, 11, 19, 12, 35, 0, 0, time.UTC),
		}, period)
	})

	// datetime.DateTime.Add saturates about 292 years from its input, so stay below that.
	t.Run("month-end anchors stay clamped centuries away", func(t *testing.T) {
		r := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   time.Date(2024, 1, 31, 15, 0, 0, 0, time.UTC),
		}

		period, err := r.GetPeriodAt(time.Date(2224, 2, 15, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		require.Equal(t, timeutil.ClosedPeriod{
			From: time.Date(2224, 1, 31, 15, 0, 0, 0, time.UTC),
			To:   time.Date(2224, 2, 29, 15, 0, 0, 0, time.UTC),
		}, period)
	})
}

func BenchmarkRecurrenceGetPeriodAt(b *testing.B) {
	at := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	anchor := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name     string
		interval datetime.ISODuration
	}{
		{name: "hourly", interval: datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)},
		{name: "daily", interval: datetime.DurationDay},
		{name: "monthly", interval: datetime.DurationMonth},
	} {
		b.Run(tc.name, func(b *testing.B) {
			r := timeutil.Recurrence{
				Interval: timeutil.RecurrenceInterval{ISODuration: tc.interval},
				Anchor:   anchor,
			}

			for b.Loop() {
				if _, err := r.GetPeriodAt(at); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
