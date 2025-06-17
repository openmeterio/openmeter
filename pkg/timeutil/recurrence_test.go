package timeutil_test

import (
	"fmt"
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

func TestPeriodGeneration(t *testing.T) {
	bpTz, err := time.LoadLocation("Europe/Budapest")
	assert.NoError(t, err)

	tc := []struct {
		name       string
		recurrence timeutil.Recurrence
		from       time.Time
		expected   []timeutil.Period

		outputTz *time.Location
	}{
		{
			name: "Should return periods for monthly recurrence",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2024-12-31T00:00:00Z"),
			},
			from: testutils.GetRFC3339Time(t, "2024-12-31T00:00:00Z"),
			expected: []timeutil.Period{
				{
					From: testutils.GetRFC3339Time(t, "2024-12-31T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-01-31T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-01-31T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-02-28T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-02-28T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-03-31T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-03-31T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-04-30T00:00:00Z"),
				},
			},
		},
		{
			// Last leap second was happening at  The most recent leap second was on December 31, 2016.
			name: "Leap year handling",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodYear,
				Anchor:   testutils.GetRFC3339Time(t, "2024-02-29T00:00:00Z"),
			},
			from: testutils.GetRFC3339Time(t, "2024-02-29T00:00:00Z"),
			expected: []timeutil.Period{
				{
					From: testutils.GetRFC3339Time(t, "2024-02-29T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-02-28T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-02-28T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2026-02-28T00:00:00Z"),
				},
			},
		},
		{
			name: "Daylight savings changes - anchor has timezone information",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-02-01T12:00:00Z").In(bpTz),
			},
			outputTz: bpTz,
			from:     testutils.GetRFC3339Time(t, "2025-02-01T12:00:00Z").In(bpTz),
			expected: []timeutil.Period{
				{
					From: testutils.GetRFC3339Time(t, "2025-02-01T13:00:00+01:00"),
					To:   testutils.GetRFC3339Time(t, "2025-03-01T13:00:00+01:00"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-03-01T13:00:00+01:00"),
					To:   testutils.GetRFC3339Time(t, "2025-04-01T13:00:00+02:00"), // Daylight savings keeps the anchor at 13:00
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-04-01T14:00:00+02:00"),
					To:   testutils.GetRFC3339Time(t, "2025-05-01T14:00:00+02:00"),
				},
			},
		},
		{
			name: "Daylight savings changes - anchor in UTC",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2025-02-01T12:00:00Z"),
			},
			from:     testutils.GetRFC3339Time(t, "2025-02-01T12:00:00Z").In(bpTz),
			outputTz: bpTz,
			expected: []timeutil.Period{
				{
					From: testutils.GetRFC3339Time(t, "2025-02-01T12:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-03-01T12:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-03-01T12:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-04-01T12:00:00Z"), // Daylight savings keeps the anchor at 13:00
				},
				{
					From: testutils.GetRFC3339Time(t, "2025-04-01T12:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2025-05-01T12:00:00Z"),
				},
			},
		},
		{
			// Last leap second was happening at  The most recent leap second was on December 31, 2016.
			name: "Leap second handling",
			recurrence: timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2016-11-30T00:00:00Z"),
			},
			from: testutils.GetRFC3339Time(t, "2016-11-30T00:00:00Z"),
			expected: []timeutil.Period{
				{
					From: testutils.GetRFC3339Time(t, "2016-11-30T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2016-12-30T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2016-12-30T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2017-01-30T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2017-01-30T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2017-02-28T00:00:00Z"),
				},
				{
					From: testutils.GetRFC3339Time(t, "2017-02-28T00:00:00Z"),
					To:   testutils.GetRFC3339Time(t, "2017-03-30T00:00:00Z"),
				},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			// v1
			firstPeriod, err := tt.recurrence.GetPeriodAt(tt.from)
			assert.NoError(err)

			tz := time.UTC
			if tt.outputTz != nil {
				tz = tt.outputTz
			}

			periods := []timeutil.Period{firstPeriod}

			for i := 1; i < len(tt.expected); i++ {
				next, err := tt.recurrence.GetPeriodAt(periods[i-1].To)
				assert.NoError(err)
				periods = append(periods, next)
			}

			// v2
			v2recurrence := timeutil.RecurrenceV2{
				Anchor: tt.recurrence.Anchor,
				Interval: timeutil.RecurrenceIntervalV2{
					Period: tt.recurrence.Interval.Period,
				},
			}

			firstPeriodV2, err := v2recurrence.GetPeriodAt(tt.from)
			assert.NoError(err)

			periodsV2 := []timeutil.Period{firstPeriodV2}

			for i := 1; i < len(tt.expected); i++ {
				next, err := v2recurrence.GetPeriodAt(periodsV2[i-1].To)
				assert.NoError(err)
				periodsV2 = append(periodsV2, next)
			}

			fmt.Printf("\n\n%s\n---------\n\n", tt.name)

			for i := range tt.expected {
				// TODO: log
				fmt.Printf("iteration[%d]:\n\texp: [%s..%s]\n\tgot: [%s..%s]\n\tv2:  [%s..%s]\n",
					i,
					tt.expected[i].From.In(tz).Format(time.RFC3339), tt.expected[i].To.In(tz).Format(time.RFC3339),
					periods[i].From.In(tz).Format(time.RFC3339), periods[i].To.In(tz).Format(time.RFC3339),
					periodsV2[i].From.In(tz).Format(time.RFC3339), periodsV2[i].To.In(tz).Format(time.RFC3339),
				)
			}
		})
	}

	assert.True(t, false)
}
