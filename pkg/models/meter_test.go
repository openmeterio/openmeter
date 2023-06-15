package models

import (
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
			meter:       &Meter{ID: "meter1", WindowSize: WindowSizeDay, Aggregation: MeterAggregationSum},
			meterValues: []*MeterValue{},
			windowSize:  windowSizePtr(WindowSizeHour),
			wantErr:     "invalid aggregation: expected window size of DAY for meter with ID meter1, but got HOUR",
		},
		{
			meter:       &Meter{ID: "meter2", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum},
			meterValues: []*MeterValue{},
			windowSize:  windowSizePtr(WindowSizeMinute),
			wantErr:     "invalid aggregation: expected window size of HOUR or DAY for meter with ID meter2, but got MINUTE",
		},
		// same resolution
		{
			meter: &Meter{ID: "meter3", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum},
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
			meter: &Meter{ID: "meter4", WindowSize: WindowSizeHour, Aggregation: MeterAggregationSum},
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
			meter: &Meter{ID: "meter5", WindowSize: WindowSizeHour, Aggregation: MeterAggregationAvg},
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
			meter: &Meter{ID: "meter6", WindowSize: WindowSizeHour, Aggregation: MeterAggregationMax},
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
			meter: &Meter{ID: "meter7", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum},
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
			meter: &Meter{ID: "meter8", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationAvg},
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
			meter: &Meter{ID: "meter9", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMin},
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
			meter: &Meter{ID: "meter10", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMax},
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
			meter: &Meter{ID: "meter11", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum},
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
			meter: &Meter{ID: "meter12", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationAvg},
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
			meter: &Meter{ID: "meter13", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMin},
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
			meter: &Meter{ID: "meter14", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationMax},
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
		// aggregate type not supported
		{
			meter:       &Meter{ID: "meter15", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationCountDistinct},
			meterValues: []*MeterValue{},
			windowSize:  windowSizePtr(WindowSizeHour),
			wantErr:     "invalid aggregation: expected window size of MINUTE for meter with ID meter15, but got HOUR",
		},
		// aggregate to total
		{
			meter: &Meter{ID: "meter16", WindowSize: WindowSizeMinute, Aggregation: MeterAggregationSum},
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
		t.Run(tt.meter.ID, func(t *testing.T) {
			got, err := tt.meter.AggregateMeterValues(tt.meterValues, tt.windowSize)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}

			assertMeterValuesEqual(t, tt.want, got)
		})
	}
}
