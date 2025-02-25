package meter_test

import (
	"errors"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestParseEvent(t *testing.T) {
	meterSum := meter.Meter{
		Namespace:     "default",
		Slug:          "m1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "api-calls",
		ValueProperty: "$.duration_ms",
		GroupBy: map[string]string{
			"method": "$.method",
			"path":   "$.path",
		},
		WindowSize: meter.WindowSizeMinute,
	}

	meterCount := meter.Meter{
		Namespace:   "default",
		Slug:        "m2",
		Aggregation: meter.MeterAggregationCount,
		EventType:   "api-calls",
		WindowSize:  meter.WindowSizeMinute,
	}

	meterUniqueCount := meter.Meter{
		Namespace:     "default",
		Slug:          "m3",
		Aggregation:   meter.MeterAggregationUniqueCount,
		EventType:     "spans",
		WindowSize:    meter.WindowSizeMinute,
		ValueProperty: "$.trace_id",
	}

	tests := []struct {
		description string
		meter       meter.Meter
		event       func(t *testing.T) event.Event
		err         error
		value       *float64
		valueStr    *string
		groupBy     map[string]string
	}{
		{
			description: "should parse event",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			value: lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse event with numeric string value",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": "100", "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			value: lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should parse count as value one",
			meter:       meterCount,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				return ev
			},
			value:   lo.ToPtr(1.0),
			groupBy: map[string]string{},
		},
		{
			description: "should parse unique count as string",
			meter:       meterUniqueCount,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("spans")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"trace_id": "test_trace_id"}`))
				require.NoError(t, err)

				return ev
			},
			valueStr: lo.ToPtr("test_trace_id"),
			groupBy:  map[string]string{},
		},
		{
			description: "should parse event with missing group by properties",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": 100}`))
				require.NoError(t, err)

				return ev
			},
			value: lo.ToPtr(100.0),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with invalid json",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{`))
				require.NoError(t, err)

				return ev
			},
			err:     errors.New("cannot unmarshal event data"),
			groupBy: map[string]string{},
		},
		{
			description: "should return error with data missing",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				return ev
			},
			err: errors.New("event data is null and missing value property"),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with data null",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")
				_ = ev.SetData(event.ApplicationJSON, nil)

				return ev
			},
			err: errors.New("event data is null and missing value property"),
			groupBy: map[string]string{
				"method": "",
				"path":   "",
			},
		},
		{
			description: "should return error with value property not found",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			err: errors.New("event data is missing value property at \"$.duration_ms\""),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property is null",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": null, "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			err: errors.New("event data value cannot be null"),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
		{
			description: "should return error when value property cannot be parsed as number",
			meter:       meterSum,
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			err: errors.New("event data value cannot be parsed as float64: not a number"),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.description, func(t *testing.T) {
			value, valueStr, groupBy, err := meter.ParseEvent(test.meter, test.event(t))

			assert.Equal(t, test.err, err)
			assert.Equal(t, test.value, value)
			assert.Equal(t, test.valueStr, valueStr)
			assert.Equal(t, test.groupBy, groupBy)
		})
	}
}
