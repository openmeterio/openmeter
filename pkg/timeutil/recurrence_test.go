package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNextAfter(t *testing.T) {
	now := time.Now().Truncate(time.Minute)

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		boundary   timeutil.Boundary
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time if its same as anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Should return time if it falls on recurrence period",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Should return next period after anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Inclusive,
			time:     now.Add(-time.Hour),
			want:     now,
		},
		{
			name: "Should return next period if anchor is in the far past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -50),
			},
			boundary: timeutil.Inclusive,
			time:     now.Add(-time.Hour),
			want:     now,
		},
		{
			name: "Should return next if anchor is in the future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Inclusive,
			time:     now.Add(-time.Hour),
			want:     now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 50),
			},
			boundary: timeutil.Inclusive,
			time:     now.Add(-time.Hour),
			want:     now,
		},
		{
			name: "Should work with weeks",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodWeek,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, 6),
		},
		{
			name: "Should work with months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   addDateNoOverflow(now, 0, 0, 0),
			},
			boundary: timeutil.Inclusive,
			time:     addDateNoOverflow(now, 0, 0, 1),
			want:     addDateNoOverflow(now, 0, 1, 0),
		},
		// Exclusive boundary corner cases
		{
			name: "Exclusive: Should return the next anchor if anchor matches t",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, 1),
		},
		{
			name: "Exclusive: Should return the next anchor t is on anchor point, anchor is in past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, 1),
		},
		{
			name: "Exclusive: Should return the next anchor t is on anchor point, anchor is in future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, 1),
		},
		// Inclusive boundary corner cases
		{
			name: "Inclusive: Should return the anchor if anchor matches t",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Inclusive: Should return the anchor if t is on anchor point, anchor is in past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Inclusive: Should return the anchor if t is on anchor point, anchor is in future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.recurrence.NextAfter(tt.time, tt.boundary)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrevBefore(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		boundary   timeutil.Boundary
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time - period if time is same as anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -1),
		},
		{
			name: "Should return time - period if time falls on recurrence period",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -1),
		},
		{
			name: "Should return prev period after anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Exclusive,
			time:     now.Add(time.Hour),
			want:     now,
		},
		{
			name: "Should return prev period if anchor is in the far past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -50),
			},
			boundary: timeutil.Exclusive,
			time:     now.Add(time.Hour),
			want:     now,
		},
		{
			name: "Should return prev if anchor is in the future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Exclusive,
			time:     now.Add(time.Hour),
			want:     now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 50),
			},
			boundary: timeutil.Exclusive,
			time:     now.Add(time.Hour),
			want:     now,
		},
		{
			name: "Should work with weeks",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodWeek,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -6),
		},
		{
			name: "Should work with months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			},
			boundary: timeutil.Exclusive,
			time:     addDateNoOverflow(now, 0, 0, 1),
			want:     now,
		},
		{
			name: "Should work on 29th of January",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-01-29T12:00:00Z"),
			},
			boundary: timeutil.Exclusive,
			time:     testutils.GetRFC3339Time(t, "2025-01-29T12:10:00Z"),
			want:     testutils.GetRFC3339Time(t, "2025-01-29T12:00:00Z"),
		},
		// Exclusive boundary corner cases
		{
			name: "Exclusive: Should return the previous anchor if anchor matches t",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -1),
		},
		{
			name: "Exclusive: Should return the previous anchor t is on anchor point, anchor is in past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -1),
		},
		{
			name: "Exclusive: Should return the previous anchor t is on anchor point, anchor is in future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Exclusive,
			time:     now,
			want:     addDateNoOverflow(now, 0, 0, -1),
		},
		// Inclusive boundary corner cases
		{
			name: "Inclusive: Should return the anchor if anchor matches t",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Inclusive: Should return the anchor if t is on anchor point, anchor is in past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
		{
			name: "Inclusive: Should return the anchor if t is on anchor point, anchor is in future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, 1),
			},
			boundary: timeutil.Inclusive,
			time:     now,
			want:     now,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.recurrence.PrevBefore(tt.time, tt.boundary)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPeriodAt(t *testing.T) {
	now := clock.Now().Truncate(time.Millisecond)

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		time       time.Time
		want       timeutil.ClosedPeriod
	}{
		{
			name: "Should return next period if time falls on recurrence period",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			time: now,
			want: timeutil.ClosedPeriod{
				From: now,
				To:   addDateNoOverflow(now, 0, 0, 1),
			},
		},
		{
			name: "Should return containing period in general case",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   addDateNoOverflow(now, 0, 0, -1),
			},
			time: now.Add(-time.Hour),
			want: timeutil.ClosedPeriod{
				From: addDateNoOverflow(now, 0, 0, -1),
				To:   now,
			},
		},
		{
			name: "Correctly handles variable length months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-01-31T15:00:00Z"),
			},
			time: testutils.GetRFC3339Time(t, "2025-08-13T20:00:00Z"),
			want: timeutil.ClosedPeriod{
				From: testutils.GetRFC3339Time(t, "2025-07-31T15:00:00Z"),
				To:   testutils.GetRFC3339Time(t, "2025-08-31T15:00:00Z"),
			},
		},
		{
			name: "Correctly handles variable t being at the end of the period (we are exclusive at the end)",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-01-31T15:00:00Z"),
			},
			time: testutils.GetRFC3339Time(t, "2025-09-30T15:00:00Z"),
			want: timeutil.ClosedPeriod{
				From: testutils.GetRFC3339Time(t, "2025-09-30T15:00:00Z"),
				To:   testutils.GetRFC3339Time(t, "2025-10-31T15:00:00Z"),
			},
		},
		{
			name: "Correctly handles variable length months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-01-30T15:00:00Z"),
			},
			time: testutils.GetRFC3339Time(t, "2025-02-13T20:00:00Z"),
			want: timeutil.ClosedPeriod{
				From: testutils.GetRFC3339Time(t, "2025-01-30T15:00:00Z"),
				To:   testutils.GetRFC3339Time(t, "2025-02-28T15:00:00Z"),
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.recurrence.GetPeriodAt(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

func addDateNoOverflow(t time.Time, years int, months int, days int) time.Time {
	return datetime.NewDateTime(t).AddDateNoOverflow(years, months, days).AsTime()
}
