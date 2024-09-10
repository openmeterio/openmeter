package meter_test

import (
	"errors"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestValidateEvent(t *testing.T) {
	m := meter.Meter{
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

	tests := []struct {
		description string
		event       func(t *testing.T) event.Event
		want        error
	}{
		{
			description: "should return error with invalid json",
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{`))
				require.NoError(t, err)

				return ev
			},
			want: errors.New("cannot unmarshal event data"),
		},
		{
			description: "should return error with value property not found",
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			want: errors.New("event data is missing value property at \"$.duration_ms\""),
		},
		{
			description: "should return error when value property is null",
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": null, "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			want: errors.New("event data value cannot be null"),
		},
		{
			description: "should return error when value property cannot be parsed as number",
			event: func(t *testing.T) event.Event {
				ev := event.New()
				ev.SetType("api-calls")

				err := ev.SetData(event.ApplicationJSON, []byte(`{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`))
				require.NoError(t, err)

				return ev
			},
			want: errors.New("event data value cannot be parsed as float64: not a number"),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.description, func(t *testing.T) {
			err := meter.ValidateEvent(m, test.event(t))
			if test.want == nil {
				assert.Nil(t, err)

				return
			}

			assert.Equal(t, test.want, err)
		})
	}
}
