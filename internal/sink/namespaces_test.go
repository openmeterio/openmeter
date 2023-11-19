package sink_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNamespaStore(t *testing.T) {
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
		namespace   string
		event       serializer.CloudEventsKafkaPayload
		want        error
	}{
		{
			description: "should return error with non existing namespace",
			namespace:   "non-existing-namespace",
			event:       serializer.CloudEventsKafkaPayload{},
			want:        sink.NewProcessingError("namespace not found: non-existing-namespace", sink.DROP),
		},
		{
			description: "should return error with corresponding meter not found",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "non-existing-event-type",
			},
			want: sink.NewProcessingError("no meter found for event type: non-existing-event-type", sink.INVALID),
		},
		{
			description: "should return error with invalid json",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "api-calls",
				Data: `{`,
			},
			want: sink.NewProcessingError("cannot unmarshal event data as json", sink.INVALID),
		},
		{
			description: "should return error with value property not found",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "api-calls",
				Data: `{"method": "GET", "path": "/api/v1"}`,
			},
			want: sink.NewProcessingError("event data is missing value property at $.duration_ms", sink.INVALID),
		},
		{
			description: "should return error when value property is null",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "api-calls",
				Data: `{"duration_ms": null, "method": "GET", "path": "/api/v1"}`,
			},
			want: sink.NewProcessingError("event data value cannot be null", sink.INVALID),
		},
		{
			description: "should return error when value property cannot be parsed as number",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "api-calls",
				Data: `{"duration_ms": "not a number", "method": "GET", "path": "/api/v1"}`,
			},
			want: sink.NewProcessingError("event data value cannot be parsed as float64: not a number", sink.INVALID),
		},
		{
			description: "should pass with valid event",
			namespace:   "default",
			event: serializer.CloudEventsKafkaPayload{
				Type: "api-calls",
				Data: `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			err := namespaces.ValidateEvent(ctx, tt.event, tt.namespace)
			if tt.want == nil {
				assert.Nil(t, err)
				return
			}
			assert.Equal(t, tt.want, err)
		})
	}
}
