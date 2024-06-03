package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestMockStreamingConnector(t *testing.T) {
	defaultMeterSlug := "default-meter"

	type tc struct {
		Name          string
		Events        []SimpleEvent
		Rows          []models.MeterQueryRow
		Query         *streaming.QueryParams
		Expected      []models.MeterQueryRow
		ExpectedError error
	}

	now, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)

	tt := []tc{
		{
			Name: "Should return error if meter not found",
			Query: &streaming.QueryParams{
				From: ToPtr(now.Add(-time.Hour)),
				To:   ToPtr(now),
			},
			ExpectedError: &models.MeterNotFoundError{MeterSlug: defaultMeterSlug},
		},
		{
			Name: "Should error if meter exists but doesnt match",
			Query: &streaming.QueryParams{
				From: ToPtr(now.Add(-time.Hour)),
				To:   ToPtr(now),
			},
			ExpectedError: &models.MeterNotFoundError{MeterSlug: defaultMeterSlug},
			Events:        []SimpleEvent{{MeterSlug: ulid.Make().String(), Value: 0, Time: now}},
		},
		{
			Name: "Should return empty rows if no rows and no events",
			Query: &streaming.QueryParams{
				From: ToPtr(now.Add(-time.Hour)),
				To:   ToPtr(now),
			},
			Expected: []models.MeterQueryRow{{
				Value:       0,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
			Rows: []models.MeterQueryRow{},
			// meter has to exist
			Events: []SimpleEvent{{MeterSlug: defaultMeterSlug, Value: 0, Time: now}},
		},
		{
			Name: "Should return exact row",
			Query: &streaming.QueryParams{
				From: ToPtr(now.Add(-time.Hour)),
				To:   ToPtr(now),
			},
			Expected: []models.MeterQueryRow{{
				Value:       1,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
			Rows: []models.MeterQueryRow{{
				Value:       1,
				WindowStart: now.Add(-time.Hour),
				WindowEnd:   now,
				GroupBy:     map[string]*string{},
			}},
		},
		{
			Name: "Should return event sum",
			Query: &streaming.QueryParams{
				From: ToPtr(now.Add(-time.Hour)),
				To:   ToPtr(now),
			},
			Expected: []models.MeterQueryRow{{
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
			Query: &streaming.QueryParams{
				From: ToPtr(now.Truncate(time.Minute).Add(time.Second * 30).Add(-time.Minute * 2)),
				To:   ToPtr(now.Truncate(time.Minute).Add(time.Second * 30)),
			},
			Expected: []models.MeterQueryRow{{
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
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			streamingConnector := NewMockStreamingConnector(t, MockStreamingConnectorParams{DefaultHighwatermark: now})

			for _, event := range tc.Events {
				streamingConnector.AddResponse(event.MeterSlug, event.Value, event.Time)
			}

			for _, row := range tc.Rows {
				streamingConnector.AddRow(defaultMeterSlug, row)
			}

			result, err := streamingConnector.QueryMeter(context.Background(), "namespace", defaultMeterSlug, tc.Query)
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
