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
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
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

	tt := []tc{
		{
			Name: "Should return error if meter not found",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			ExpectedError: meter.NewMeterNotFoundError(defaultMeterSlug),
		},
		{
			Name: "Should error if meter exists but doesnt match",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			ExpectedError: meter.NewMeterNotFoundError(defaultMeterSlug),
			Events: []SimpleEvent{
				{MeterSlug: ulid.Make().String(), Value: 0, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
			},
		},
		{
			Name: "Should return empty rows if no rows and no events",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			Expected: []meter.MeterQueryRow{},
			Rows:     []meter.MeterQueryRow{},
			// meter has to exist
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 0, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
			},
		},
		{
			Name: "Should return exact row",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			Expected: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Rows: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
		},
		{
			Name: "Should return event sum",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			Expected: []meter.MeterQueryRow{{
				Value:       5,
				WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
			},
		},
		{
			Name: "Should aggregate events as if they were windowed - minute window",
			Query: streaming.QueryParams{
				From:       convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:58:30Z")),
				To:         convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:30Z")),
				WindowSize: convert.ToPointer(meter.WindowSizeMinute), // Force 1 minute window (even if this is not a valid api call on the main api)
			},
			Expected: []meter.MeterQueryRow{{
				Value:       2,
				WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:58:30Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:30Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// period start
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:30Z")},
				// event in window of periodstart but before it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:29Z")},
				// event in window of periodstart but after it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:31Z")},
				// event in only valid window at start of it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				// event in only valid window in middle of it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:30Z")},
				// For simple event aggregation end is exclusive so this should not count
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// event in window of periodend but before it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:29Z")},
				// period end
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:30Z")},
				// event in window of periodend but after it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:31Z")},
			},
		},
		// Second window size
		{
			Name: "Should aggregate events as if they were windowed - second window",
			Query: streaming.QueryParams{
				From:       convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00.1234Z")),
				To:         convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:30.1234Z")),
				WindowSize: convert.ToPointer(meter.WindowSizeSecond), // Force 1 minute window (even if this is not a valid api call on the main api)
			},
			Expected: []meter.MeterQueryRow{{
				Value:       3,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:30Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// event before periodstart
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
				// period start
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// event after periodstart
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:01.11Z")},
				// event in window of periodend but before it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:29.123Z")},
				// period end
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:30Z")},
				// For simple event aggregation end is exclusive so this should not count
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:30.123Z")},
				// event in window of periodend but after it
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:31Z")},
			},
		},
		{
			Name: "Should return events windowed",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z")),
				To:             convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:58:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       2,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:58:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:57:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:02Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
			},
		},
		{
			Name: "Should return events windowed even if query from and to don't align with window boundaries",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:57:01Z")),
				To:             convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:58:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       2,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:58:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:57:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:02Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
			},
		},
		{
			Name: "Should not return rows for periods in which there are no events",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:57:01Z")),
				To:             convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")),
				WindowSize:     convert.ToPointer(meter.WindowSizeMinute),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       1,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:58:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:57:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
			},
		},
		{
			Name: "Should return row for queried period if window is larger than period if there are events in the period",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z")),
				To:             convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				WindowSize:     convert.ToPointer(meter.WindowSizeHour),
				WindowTimeZone: time.UTC,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       8,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:57:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:02Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
			},
		},
		{
			Name: "Should use latest value if meter.Aggregation is LATEST when NOT windowed",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
			},
			Meter: meter.Meter{
				Key:         defaultMeterSlug,
				Aggregation: meter.MeterAggregationLatest,
			},
			Expected: []meter.MeterQueryRow{
				{
					Value:       3,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:57:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				// Events should be sorted by time ASC and the LAST value should be returned
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:00Z")},
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:57:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:58:02Z")},
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
				{MeterSlug: defaultMeterSlug, Value: 4, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:02Z")},
			},
		},
		// FilterStoredAt tests
		{
			Name: "StoredAt defaults to Time when no WithStoredAt option is provided",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")),
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						// Include only events stored at or before 00:00; default StoredAt == Time
						Lte: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
					},
				},
			},
			Expected: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// No StoredAt set → AddSimpleEvent defaults it to Time (00:00), passes Lte filter
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// No StoredAt set → defaults to Time (00:30) which is > 00:00, excluded by filter
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z")},
			},
		},
		{
			Name: "Should filter events by FilterStoredAt using Lt predicate with WithStoredAt override",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")),
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						Lt: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z")),
					},
				},
			},
			Expected: []meter.MeterQueryRow{{
				Value:       1,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// StoredAt 00:00 < Lt 00:30 → included
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// StoredAt 01:00 is NOT < Lt 00:30 → excluded
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")},
			},
		},
		{
			Name: "Should filter events by FilterStoredAt using Gte predicate with WithStoredAt override",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")),
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						Gte: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")),
					},
				},
			},
			Expected: []meter.MeterQueryRow{{
				Value:       2,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// StoredAt 00:00 < Gte 01:00 → excluded
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// StoredAt 01:00 >= Gte 01:00 → included
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")},
			},
		},
		{
			Name: "Should filter events by FilterStoredAt using $and composite predicate",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z")),
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						And: &[]filter.FilterTime{
							// StoredAt >= 00:30 AND StoredAt < 01:30 → only middle event
							{Gte: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"))},
							{Lt: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:30:00Z"))},
						},
					},
				},
			},
			Expected: []meter.MeterQueryRow{{
				Value:       2,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// StoredAt 00:00: fails Gte 00:30 → excluded
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// StoredAt 01:00: passes both Gte 00:30 and Lt 01:30 → included
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")},
				// StoredAt 02:00: fails Lt 01:30 → excluded
				{MeterSlug: defaultMeterSlug, Value: 4, Time: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z")},
			},
		},
		{
			Name: "Should filter events by FilterStoredAt using $or composite predicate",
			Query: streaming.QueryParams{
				From: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
				To:   convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z")),
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						Or: &[]filter.FilterTime{
							// StoredAt < 00:30 OR StoredAt >= 01:30 → first and last events
							{Lt: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"))},
							{Gte: convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T01:30:00Z"))},
						},
					},
				},
			},
			Expected: []meter.MeterQueryRow{{
				Value:       5,
				WindowStart: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
				WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z"),
				GroupBy:     map[string]*string{},
			}},
			Events: []SimpleEvent{
				// StoredAt 00:00: passes Lt 00:30 → included
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")},
				// StoredAt 01:00: fails both arms → excluded
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2024-01-01T00:30:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z")},
				// StoredAt 02:00: passes Gte 01:30 → included
				{MeterSlug: defaultMeterSlug, Value: 4, Time: testutils.GetRFC3339Time(t, "2024-01-01T01:00:00Z"), StoredAt: testutils.GetRFC3339Time(t, "2024-01-01T02:00:00Z")},
			},
		},
		{
			Name: "Should use latest value if meter.Aggregation is LATEST when windowed",
			Query: streaming.QueryParams{
				From:           convert.ToPointer(testutils.GetRFC3339Time(t, "2023-12-31T22:00:00Z")),
				To:             convert.ToPointer(testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z")),
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
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T22:00:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
					GroupBy:     map[string]*string{},
				},
				{
					Value:       5,
					WindowStart: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z"),
					WindowEnd:   testutils.GetRFC3339Time(t, "2024-01-01T00:00:00Z"),
					GroupBy:     map[string]*string{},
				},
			},
			Events: []SimpleEvent{
				{MeterSlug: defaultMeterSlug, Value: 1, Time: testutils.GetRFC3339Time(t, "2023-12-31T21:59:58Z")}, // Should be ignored
				{MeterSlug: defaultMeterSlug, Value: 3, Time: testutils.GetRFC3339Time(t, "2023-12-31T22:00:06Z")}, // Should be last value in first window
				{MeterSlug: defaultMeterSlug, Value: 2, Time: testutils.GetRFC3339Time(t, "2023-12-31T22:00:02Z")},
				{MeterSlug: defaultMeterSlug, Value: 4, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:00:00Z")}, // Should fall into second window
				{MeterSlug: defaultMeterSlug, Value: 5, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:59Z")},
				{MeterSlug: defaultMeterSlug, Value: 6, Time: testutils.GetRFC3339Time(t, "2023-12-31T23:59:58Z")},
				{MeterSlug: defaultMeterSlug, Value: 7, Time: testutils.GetRFC3339Time(t, "2024-01-01T23:00:02Z")}, // Should be ignored
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			streamingConnector := NewMockStreamingConnector(t)

			for _, event := range tc.Events {
				var opts []AddOption
				if !event.StoredAt.IsZero() {
					opts = append(opts, WithStoredAt(event.StoredAt))
				}
				streamingConnector.AddSimpleEvent(event.MeterSlug, event.Value, event.Time, opts...)
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
