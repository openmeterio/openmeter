package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func windowSizePtr(ws WindowSize) *WindowSize {
	return &ws
}

func assertMeterValuesEqual(t *testing.T, want, got []*MeterValue) {
	assert.Equal(t, len(want), len(got))
	for i := range want {
		assert.Equal(t, *want[i], *got[i])
	}
}

func TestAggregateMeterValues(t *testing.T) {
	baseTime, _ := time.Parse(time.RFC3339, "2021-02-03T01:02:00.000Z")
	minuteWindows := []struct {
		start time.Time
		end   time.Time
	}{{
		start: baseTime.Truncate(time.Minute),
		end:   baseTime.Add(time.Minute).Truncate(time.Minute),
	}, {
		start: baseTime.Add(2 * time.Minute).Truncate(time.Minute),
		end:   baseTime.Add(3 * time.Minute).Truncate(time.Minute),
	}, {
		start: baseTime.Add(time.Hour).Truncate(time.Minute),
		end:   baseTime.Add(time.Hour + time.Minute).Truncate(time.Minute),
	}, {
		start: baseTime.Add(2 * time.Hour).Truncate(time.Minute),
		end:   baseTime.Add(2*time.Hour + time.Minute).Truncate(time.Minute),
	}, {
		start: baseTime.Add(48 * time.Hour).Truncate(time.Minute),
		end:   baseTime.Add(48*time.Hour + time.Minute).Truncate(time.Minute),
	}, {
		start: baseTime.Add(49 * time.Hour).Truncate(time.Minute),
		end:   baseTime.Add(49*time.Hour + time.Minute).Truncate(time.Minute),
	}}
	hourWindows := []struct {
		start time.Time
		end   time.Time
	}{{
		start: baseTime.Truncate(time.Hour),
		end:   baseTime.Add(time.Hour).Truncate(time.Hour),
	}, {
		start: baseTime.Add(2 * time.Hour).Truncate(time.Hour),
		end:   baseTime.Add(3 * time.Hour).Truncate(time.Hour),
	}, {
		start: baseTime.Add(3 * time.Hour).Truncate(time.Hour),
		end:   baseTime.Add(2 * time.Hour).Truncate(time.Hour),
	}}

	tests := []struct {
		meter       *Meter
		meterValues []*MeterValue
		windowSize  *WindowSize
		want        []*MeterValue
		wantErr     string
	}{
		// aggregate to lower resolution
		{
			meter:       &Meter{Slug: "meter1", WindowSize: WindowSizeDay, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{},
			windowSize:  windowSizePtr(WindowSizeHour),
			wantErr:     "invalid aggregation: expected window size of DAY, but got HOUR",
		},
		{
			meter:       &Meter{Slug: "meter2", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{},
			windowSize:  windowSizePtr(WindowSizeMinute),
			wantErr:     "invalid aggregation: expected window size of HOUR or DAY, but got MINUTE",
		},
		// same resolution
		{
			meter: &Meter{Slug: "meter3", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 200},
			},
			windowSize: windowSizePtr(WindowSizeHour),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 200},
			},
		},
		// aggregate to higher resolution
		// hour -> day
		{
			meter: &Meter{Slug: "meter4", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 300},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 700},
			},
		},
		{
			meter: &Meter{Slug: "meter5", WindowSize: WindowSizeHour, Aggregation: MeterAggregationAvg, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 150},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 350},
			},
		},
		{
			meter: &Meter{Slug: "meter6", WindowSize: WindowSizeHour, Aggregation: MeterAggregationMax, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: hourWindows[0].start, WindowEnd: hourWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: hourWindows[1].start, WindowEnd: hourWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 200},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 400},
			},
		},
		// aggregate to higher resolution
		// minute -> hour
		{
			meter: &Meter{Slug: "meter7", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeHour),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 300},
				{Subject: "s2", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 700},
			},
		},
		{
			meter: &Meter{Slug: "meter8", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationAvg, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeHour),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 150},
				{Subject: "s2", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 350},
			},
		},
		{
			meter: &Meter{Slug: "meter9", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMin, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeHour),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 100},
				{Subject: "s2", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 300},
			},
		},
		{
			meter: &Meter{Slug: "meter10", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMax, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeHour),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 200},
				{Subject: "s2", WindowStart: baseTime.Truncate(time.Hour), WindowEnd: baseTime.Truncate(time.Hour).Add(time.Hour), Value: 400},
			},
		},
		// aggregate to higher resolution
		// minute -> day
		{
			meter: &Meter{Slug: "meter11", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[3].start, WindowEnd: minuteWindows[3].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 300},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 700},
			},
		},
		{
			meter: &Meter{Slug: "meter12", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationAvg, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[3].start, WindowEnd: minuteWindows[3].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 150},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 350},
			},
		},
		{
			meter: &Meter{Slug: "meter13", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMin, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[3].start, WindowEnd: minuteWindows[3].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 100},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 300},
			},
		},
		{
			meter: &Meter{Slug: "meter14", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMax, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s2", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 300},
				{Subject: "s2", WindowStart: minuteWindows[3].start, WindowEnd: minuteWindows[3].end, Value: 400},
			},
			windowSize: windowSizePtr(WindowSizeDay),
			want: []*MeterValue{
				{Subject: "s1", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 200},
				{Subject: "s2", WindowStart: baseTime.Truncate(24 * time.Hour), WindowEnd: baseTime.Truncate(24 * time.Hour).Add(24 * time.Hour), Value: 400},
			},
		},
		// aggregate to higher resolution
		// aggregate to total
		{
			meter: &Meter{Slug: "meter16", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
			meterValues: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 100},
				{Subject: "s1", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 200},
				{Subject: "s1", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 300},
				{Subject: "s1", WindowStart: minuteWindows[3].start, WindowEnd: minuteWindows[3].end, Value: 400},
				{Subject: "s1", WindowStart: minuteWindows[4].start, WindowEnd: minuteWindows[4].end, Value: 500},
				{Subject: "s1", WindowStart: minuteWindows[5].start, WindowEnd: minuteWindows[5].end, Value: 600},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[0].end, Value: 700},
				{Subject: "s2", WindowStart: minuteWindows[1].start, WindowEnd: minuteWindows[1].end, Value: 800},
				{Subject: "s2", WindowStart: minuteWindows[2].start, WindowEnd: minuteWindows[2].end, Value: 900},
			},
			windowSize: nil,
			want: []*MeterValue{
				{Subject: "s1", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[5].end, Value: 2100},
				{Subject: "s2", WindowStart: minuteWindows[0].start, WindowEnd: minuteWindows[2].end, Value: 2400},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.meter.Slug, func(t *testing.T) {
			err := tt.meter.Validate()
			if err != nil {
				t.Fatal(err)
			}

			got, err := AggregateMeterValues(tt.meterValues, tt.meter.Aggregation, tt.windowSize)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}

			assertMeterValuesEqual(t, tt.want, got)
		})
	}
}

func TestWindowSizeFromDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  WindowSize
		error error
	}{
		{
			input: time.Minute,
			want:  WindowSizeMinute,
			error: nil,
		},
		{
			input: time.Hour,
			want:  WindowSizeHour,
			error: nil,
		},
		{
			input: 24 * time.Hour,
			want:  WindowSizeDay,
			error: nil,
		},
		{
			input: 2 * time.Minute,
			want:  "",
			error: fmt.Errorf("invalid window size duration: %s", 2*time.Minute),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := WindowSizeFromDuration(tt.input)
			if err != nil {
				if tt.error == nil {
					t.Error(err)
				}

				assert.Equal(t, tt.error, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
