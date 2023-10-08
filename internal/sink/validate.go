package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/oliveagle/jsonpath"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type NamespaceStore struct {
	Meters []*models.Meter
}

type SinkState struct {
	running      bool
	messageCount int
	buffer       []SinkMessage
	lastSink     time.Time
	namespaces   map[string]*NamespaceStore
}

// validateEvent validates a single event by matching it with the corresponding meter if any
func (a *SinkState) validateEvent(ctx context.Context, event serializer.CloudEventsKafkaPayload, namespace string) error {
	namespaceStore := a.namespaces[namespace]
	if namespaceStore == nil {
		// We drop events from unknown org
		return NewProcessingError(fmt.Sprintf("namespace not found: %s", namespace), DROP)
	}

	// Validate a single event against multiple meters
	found := 0
	for _, meter := range namespaceStore.Meters {
		if meter.EventType == event.Type {
			found++
			err := validateEventWithMeter(meter, event)
			if err != nil {
				return err
			}
			// A single event can match multiple meters so we cannot break the loop early
		}
	}

	if found == 0 {
		// Send to dead letter queue so we can show it to the user
		return NewProcessingError(fmt.Sprintf("no meter found for event type: %s", event.Type), DEADLETTER)
	}

	return nil
}

// validateEventWithMeter validates a single event against a single meter
func validateEventWithMeter(meter *models.Meter, ev serializer.CloudEventsKafkaPayload) *ProcessingError {
	// We can skip count events with no group bys
	if meter.Aggregation != models.MeterAggregationCount && len(meter.GroupBy) == 0 {
		return nil
	}

	// Parse CloudEvents data as JSON, currently we only support JSON encoding
	var data interface{}
	err := json.Unmarshal([]byte(ev.Data), &data)
	if err != nil {
		return NewProcessingError("cannot unmarshal event data as json", DEADLETTER)
	}

	// Parse value
	if meter.Aggregation != models.MeterAggregationCount {
		valueRaw, err := jsonpath.JsonPathLookup(data, meter.ValueProperty)
		if err != nil {
			return NewProcessingError(fmt.Sprintf("event data is missing value property at %s", meter.ValueProperty), DEADLETTER)
		}
		if valueRaw == nil {
			return NewProcessingError("event data value cannot be null", DEADLETTER)
		}

		if valueStr, ok := valueRaw.(string); ok {
			_, err = strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return NewProcessingError(fmt.Sprintf("event data value cannot be parsed as float64: %s", valueStr), DEADLETTER)
			}
		} else if _, ok := valueRaw.(float64); ok {

		} else {
			return NewProcessingError("event data value property cannot be parsed", DEADLETTER)
		}
	}

	// Parse group bys
	for _, groupByJsonPath := range meter.GroupBy {
		groupByValue, err := jsonpath.JsonPathLookup(data, groupByJsonPath)
		if err != nil {
			return NewProcessingError(fmt.Sprintf("event data is missing the group by property at %s", groupByJsonPath), DEADLETTER)
		}
		if groupByValue == nil {
			return NewProcessingError(fmt.Sprintf("event data group by property is nil at %s", groupByJsonPath), DEADLETTER)
		}
	}

	return nil
}
