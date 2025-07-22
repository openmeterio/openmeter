package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func TestMockStreamingConnector(t *testing.T) {
	defaultMeterSlug := "default-meter"

	defaultMeter := meter.Meter{
		Key:         defaultMeterSlug,
		Aggregation: meter.MeterAggregationSum,
	}

	type tc struct {
		Name          string
		Meter         meter.Meter
		Events        []SimpleEvent
		Rows          []meter.MeterQueryRow
		Query         streaming.QueryParams
		Expected      []meter.MeterQueryRow
		ExpectedError error
	}

	now, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)

	tt := []tc{
		{
			Name: "Should return error if meter not found",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Hour)),
				To:   convert.ToPointer(now),
			},
			ExpectedError: meter.NewMeterNotFoundError(defaultMeterSlug),
		},
		{
			Name: "Should error if meter exists but doesnt match",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Hour)),
				To:   convert.ToPointer(now),
			},
			ExpectedError: meter.NewMeterNotFoundError(defaultMeterSlug),
			Events:        []SimpleEvent{{MeterSlug: ulid.Make().String(), Value: 0, Time: now}},
		},
		{
			Name: "Should return empty rows if no rows and no events",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Hour)),
				To:   convert.ToPointer(now),
			},
			Expected: []meter.MeterQueryRow{},
			Rows:     []meter.MeterQueryRow{},
			// meter has to exist
			Events: []SimpleEvent{{MeterSlug: defaultMeterSlug, Value: 0, Time: now}},
		},
		{
			Name: "Should return exact row",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Hour)),
				To:   convert.ToPointer(now),
			},
			Expected: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
			Rows: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
		},
		{
			Name: "Should return event sum",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Hour)),
				To:   convert.ToPointer(now),
			},
			Expected: []meter.MeterQueryRow{{
				Value:       5,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
			},
		},
		{
			Name: "Should aggregate events as if they were windowed",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Truncate(time.Minute).Add(time.Second * 30).Add(-time.Minute * 2)),
				To:   convert.ToPointer(now.Truncate(time.Minute).Add(time.Second * 30)),
			},
			Expected: []meter.MeterQueryRow{{
				Value:       2,
				WindowStart: now.Truncate(time.Minute).Add(time.Second * 30).Add(-time.Minute * 2),
				WindowEnd:   now.Truncate(time.Minute).Add(time.Second * 30),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// period start
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 30).Add(-time.Minute * 2)},
				// event in window of periodstart but before it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 29).Add(-time.Minute * 2)},
				// event in window of periodstart but after it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 31).Add(-time.Minute * 2)},
				// event in only valid window at start of it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(-time.Minute)},
				// event in only valid window in middle of it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(-time.Minute).Add(time.Second * 30)},
				// For simple event aggregation end is exclusive so this should not count
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute)},
				// event in window of periodend but before it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 29)},
				// period end
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 30)},
				// event in window of periodend but after it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Truncate(time.Minute).Add(time.Second * 31)},
			},
		},
		{
			Name: "Should return events windowed",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(now.Add(-time.Minute * 3)),
				To:             convert.ToPointer(now),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: now.Add(-time.Minute * 3),
					WindowEnd:   now.Add(-time.Minute * 2),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       2,
					WindowStart: now.Add(-time.Minute * 2),
					WindowEnd:   now.Add(-time.Minute),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: now.Add(-time.Minute),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Minute * 2).Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute * 2).Add(time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
			},
		},
		{
			Name: "Should return events windowed even if query from and to don't align with window boundaries",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(now.Add(-time.Minute * 3).Add(time.Second)),
				To:             convert.ToPointer(now.Add(-time.Second)),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: now.Add(-time.Minute * 3),
					WindowEnd:   now.Add(-time.Minute * 2),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       2,
					WindowStart: now.Add(-time.Minute * 2),
					WindowEnd:   now.Add(-time.Minute),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: now.Add(-time.Minute),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Minute * 2).Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute * 2).Add(time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
			},
		},
		{
			Name: "Should not return rows for periods in which there are no events",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(now.Add(-time.Minute * 3).Add(time.Second)),
				To:             convert.ToPointer(now.Add(-time.Second)),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: now.Add(-time.Minute * 3),
					WindowEnd:   now.Add(-time.Minute * 2),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: now.Add(-time.Minute),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Minute * 2).Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
			},
		},
		{
			Name: "Should return row for queried period if window is larger than period if there are events in the period",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(now.Add(-time.Minute * 3)),
				To:             convert.ToPointer(now),
				WindowSize:     convert.ToPointer(meter.WindowSizeHour),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       8,
					WindowStart: now.Add(-time.Hour),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Minute * 2).Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute * 2).Add(time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
			},
		},
		{
			Name: "Should use latest value if meter.Aggregation is LATEST when NOT windowed",
			Query: streaming.QueryParams{
				From: convert.ToPointer(now.Add(-time.Minute * 3)),
				To:   convert.ToPointer(now),
			},
			Meter: meter.Meter{
				Key:         defaultMeterSlug,
				Aggregation: meter.MeterAggregationLatest,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       3,
					WindowStart: now.Add(-time.Minute * 3),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				// Events should be sorted by time ASC and the LAST value should be returned
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute)},
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Minute * 2).Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Minute * 2).Add(time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Second)},
				{MeterSlug: defaultMeterSlug, Value: 4, Time: now.Add(time.Second * 2)},
			},
		},
		{
			Name: "Should use latest value if meter.Aggregation is LATEST when windowed",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(now.Add(-time.Hour * 2)),
				To:             convert.ToPointer(now),
				WindowSize:     convert.ToPointer(meter.WindowSizeHour),
				WindowTimeZone: time.UTC,
			},
			Meter: meter.Meter{
				Key:         defaultMeterSlug,
				Aggregation: meter.MeterAggregationLatest,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       3,
					WindowStart: now.Add(-time.Hour * 2),
					WindowEnd:   now.Add(-time.Hour),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: now.Add(-time.Hour),
					WindowEnd:   now,
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: now.Add(-time.Hour * 2).Add(-time.Second * 2)}, // Should be ignored
				{MeterSlug: defaultMeterSlug, Value: 3, Time: now.Add(-time.Hour * 2).Add(time.Second * 6)},  // Should be last value in first window
				{MeterSlug: defaultMeterSlug, Value: 2, Time: now.Add(-time.Hour * 2).Add(time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 4, Time: now.Add(-time.Hour)}, // Should fall into second window
				{MeterSlug: defaultMeterSlug, Value: 5, Time: now.Add(-time.Second)},
				{MeterSlug: defaultMeterSlug, Value: 6, Time: now.Add(-time.Second * 2)},
				{MeterSlug: defaultMeterSlug, Value: 7, Time: now.Add(time.Second * 2)}, // Should be ignored
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			streamingConnector := NewMockStreamingConnector(t)

			for _, event := range tc.Events {
				streamingConnector.AddSimpleEvent(event.MeterSlug, event.Value, event.Time)
			}

			for _, row := range tc.Rows {
				streamingConnector.AddRow(defaultMeterSlug, row)
			}

			mm := lo.Ternary(tc.Meter.Key == "", defaultMeter, tc.Meter)

			result, err := streamingConnector.QueryMeter(context.Background(), "namespace", mm, tc.Query)
			if tc.ExpectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.ExpectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, result)
			}
		})
	}
}
