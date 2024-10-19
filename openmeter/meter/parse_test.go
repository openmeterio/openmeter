package meter_test

import (
	"errors"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestParseEvent(t *testing.T) {
	meterSum := meter.Meter{
		Namespace:     "default",
		Slug:          "m1",
		Description:   "",
		Aggregation:   "SUM",
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
		Description: "",
		Aggregation: "COUNT",
		EventType:   "api-calls",
		WindowSize:  meter.WindowSizeMinute,
	}

	tests := []struct {
		description string
		meter       meter.Meter
		event       func(t *testing.T) event.Event
		want        error
		value       float64
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
			value: 100,
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
			value:   1,
			groupBy: map[string]string{},
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
			value:   100,
			groupBy: map[string]string{},
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
			want:    errors.New("cannot unmarshal event data"),
			groupBy: map[string]string{},
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
			want: errors.New("event data is missing value property at \"$.duration_ms\""),
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
			want: errors.New("event data value cannot be null"),
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
			want: errors.New("event data value cannot be parsed as float64: not a number"),
			groupBy: map[string]string{
				"method": "GET",
				"path":   "/api/v1",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.description, func(t *testing.T) {
			value, groupBy, err := meter.ParseEvent(test.meter, test.event(t))
			if test.want == nil {
				assert.Nil(t, err)

				return
			}

			assert.Equal(t, test.want, err)
			assert.Equal(t, test.value, value)
			assert.Equal(t, test.groupBy, groupBy)
		})
	}
}
