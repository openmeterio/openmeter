package sink_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/sink"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNamespaceStore(t *testing.T) {
	ctx := context.Background()
	namespaces := sink.NewNamespaceStore()

	meter1 := models.Meter{
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
		WindowSize: models.WindowSizeMinute,
	}

	namespaces.AddMeter(meter1)

	tests := []struct {
		description string
		event       sinkmodels.SinkMessage
		want        sinkmodels.ProcessingStatus
	}{
		{
			description: "should return error with non existing namespace",
			event: sinkmodels.SinkMessage{
				Namespace:  "non-existing-namespace",
				Serialized: &serializer.CloudEventsKafkaPayload{},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.DROP,
				Error: errors.New("namespace not found: non-existing-namespace"),
			},
		},
		{
			description: "should return error with corresponding meter not found",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "non-existing-event-type",
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("no meter found for event type: non-existing-event-type"),
			},
		},
		{
			description: "should return error with invalid json",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "api-calls",
					Data: `{`,
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("cannot unmarshal event data as json"),
			},
		},
		{
			description: "should return error with value property not found",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "api-calls",
					Data: `{"method": "GET", "path": "/api/v1"}`,
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("event data is missing value property at $.duration_ms"),
			},
		},
		{
			description: "should return error when value property is null",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "api-calls",
					Data: `{"duration_ms": null, "method": "GET", "path": "/api/v1"}`,
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("event data value cannot be null"),
			},
		},
		{
			description: "should return error when value property cannot be parsed as number",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "api-calls",
					Data: `{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`,
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("event data value cannot be parsed as float64: not a number"),
			},
		},
		{
			description: "should pass with valid event",
			event: sinkmodels.SinkMessage{
				Namespace: "default",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Type: "api-calls",
					Data: `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
				},
			},
			want: sinkmodels.ProcessingStatus{
				State: sinkmodels.OK,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			namespaces.ValidateEvent(ctx, &tt.event)
			if tt.want.Error == nil {
				assert.Nil(t, tt.event.Status.Error)
				return
			}
			assert.Equal(t, tt.want, tt.event.Status)
		})
	}
}
