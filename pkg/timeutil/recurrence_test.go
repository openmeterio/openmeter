package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNextAfter(t *testing.T) {
	now := time.Now().Truncate(time.Minute)

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time if its same as anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			time: now,
			want: now,
		},
		{
			name: "Should return time if it falls on recurrence period",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now,
		},
		{
			name: "Should return next period after anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next period if anchor is in the far past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -50),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 50),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should work with weeks",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodWeek,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now.AddDate(0, 0, 6),
		},
		{
			name: "Should work with months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now.AddDate(0, 0, 0),
			},
			time: now.AddDate(0, 0, 1),
			want: now.AddDate(0, 1, 0),
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.recurrence.NextAfter(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrevBefore(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time - period if time is same as anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			},
			time: now,
			want: now.AddDate(0, 0, -1),
		},
		{
			name: "Should return time - period if time falls on recurrence period",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now.AddDate(0, 0, -1),
		},
		{
			name: "Should return prev period after anchor",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now.Add(+time.Hour),
			want: now,
		},
		{
			name: "Should return prev period if anchor is in the far past",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -50),
			},
			time: now.Add(+time.Hour),
			want: now,
		},
		{
			name: "Should return prev if anchor is in the future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now.Add(time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 50),
			},
			time: now.Add(time.Hour),
			want: now,
		},
		{
			name: "Should work with weeks",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodWeek,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now,
			want: now.AddDate(0, 0, -6),
		},
		{
			name: "Should work with months",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   now,
			},
			time: now.AddDate(0, 0, 1),
			want: now,
		},
		{
			name: "Should work on 29th of January",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-01-29T12:00:00Z"),
			},
			time: testutils.GetRFC3339Time(t, "2025-01-29T12:10:00Z"),
			want: testutils.GetRFC3339Time(t, "2025-01-29T12:00:00Z"),
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.recurrence.PrevBefore(tt.time)
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
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: timeutil.ClosedPeriod{
				From: now,
				To:   now.AddDate(0, 0, 1),
			},
		},
		{
			name: "Should return containing period in general case",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now.Add(-time.Hour),
			want: timeutil.ClosedPeriod{
				From: now.AddDate(0, 0, -1),
				To:   now,
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
