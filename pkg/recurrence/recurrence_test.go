package recurrence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextAfter(t *testing.T) {
	now := time.Now().Truncate(time.Minute)

	tc := []struct {
		name       string
		recurrence Recurrence
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time if its same as anchor",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now,
			},
			time: now,
			want: now,
		},
		{
			name: "Should return time if it falls on recurrence period",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now,
		},
		{
			name: "Should return next period after anchor",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next period if anchor is in the far past",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -50),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the future",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 50),
			},
			time: now.Add(-time.Hour),
			want: now,
		},
		{
			name: "Should work with weeks",
			recurrence: Recurrence{
				Interval: RecurrencePeriodWeek,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now.AddDate(0, 0, 6),
		},
		{
			name: "Should work with months",
			recurrence: Recurrence{
				Interval: RecurrencePeriodMonth,
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
		recurrence Recurrence
		time       time.Time
		want       time.Time
	}{
		{
			name: "Should return time - period if time is same as anchor",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now,
			},
			time: now,
			want: now.AddDate(0, 0, -1),
		},
		{
			name: "Should return time - period if time falls on recurrence period",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now,
			want: now.AddDate(0, 0, -1),
		},
		{
			name: "Should return prev period after anchor",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -1),
			},
			time: now.Add(+time.Hour),
			want: now,
		},
		{
			name: "Should return prev period if anchor is in the far past",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, -50),
			},
			time: now.Add(+time.Hour),
			want: now,
		},
		{
			name: "Should return prev if anchor is in the future",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now.Add(time.Hour),
			want: now,
		},
		{
			name: "Should return next if anchor is in the far future",
			recurrence: Recurrence{
				Interval: RecurrencePeriodDaily,
				Anchor:   now.AddDate(0, 0, 50),
			},
			time: now.Add(time.Hour),
			want: now,
		},
		{
			name: "Should work with weeks",
			recurrence: Recurrence{
				Interval: RecurrencePeriodWeek,
				Anchor:   now.AddDate(0, 0, 1),
			},
			time: now,
			want: now.AddDate(0, 0, -6),
		},
		{
			name: "Should work with months",
			recurrence: Recurrence{
				Interval: RecurrencePeriodMonth,
				Anchor:   now,
			},
			time: now.AddDate(0, 0, 1),
			want: now,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.recurrence.PrevBefore(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}
