package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/oliveagle/jsonpath"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type MeterStore struct {
	Meters []models.Meter
}

type NamespaceStore struct {
	namespaces map[string]*MeterStore
}

func NewNamespaceStore() *NamespaceStore {
	return &NamespaceStore{
		namespaces: make(map[string]*MeterStore),
	}
}

func (n *NamespaceStore) AddMeter(meter models.Meter) {
	if n.namespaces[meter.Namespace] == nil {
		n.namespaces[meter.Namespace] = &MeterStore{
			Meters: []models.Meter{meter},
		}
	} else {
		n.namespaces[meter.Namespace].Meters = append(n.namespaces[meter.Namespace].Meters, meter)
	}
}

// ValidateEvent validates a single event by matching it with the corresponding meter if any
func (a *NamespaceStore) ValidateEvent(ctx context.Context, event serializer.CloudEventsKafkaPayload, namespace string) ([]*models.Meter, error) {
	namespaceStore := a.namespaces[namespace]
	if namespaceStore == nil {
		// We drop events from unknown org
		return nil, NewProcessingError(fmt.Sprintf("namespace not found: %s", namespace), DROP)
	}

	// Validate a single event against multiple meters
	affectedMeters := []*models.Meter{}
	for idx, meter := range namespaceStore.Meters {
		meter := meter
		if meter.EventType == event.Type {
			affectedMeters = append(affectedMeters, &namespaceStore.Meters[idx])
			err := validateEventWithMeter(meter, event)
			if err != nil {
				return nil, err
			}
			// A single event can match multiple meters so we cannot break the loop early
		}
	}

	if len(affectedMeters) == 0 {
		// Mark as invalid so we can show it to the user
		return nil, NewProcessingError(fmt.Sprintf("no meter found for event type: %s", event.Type), INVALID)
	}

	return affectedMeters, nil
}

// validateEventWithMeter validates a single event against a single meter
func validateEventWithMeter(meter models.Meter, ev serializer.CloudEventsKafkaPayload) *ProcessingError {
	// Parse CloudEvents data as JSON, currently we only support JSON encoding
	var data interface{}
	err := json.Unmarshal([]byte(ev.Data), &data)
	if err != nil {
		return NewProcessingError("cannot unmarshal event data as json", INVALID)
	}

	// We can skip count events as they don't have value property
	if meter.Aggregation == models.MeterAggregationCount {
		return nil
	}

	// Get value from event data by value property
	valueRaw, err := jsonpath.JsonPathLookup(data, meter.ValueProperty)
	if err != nil {
		return NewProcessingError(fmt.Sprintf("event data is missing value property at %s", meter.ValueProperty), INVALID)
	}
	if valueRaw == nil {
		return NewProcessingError("event data value cannot be null", INVALID)
	}

	// Aggregation specific value validation
	switch meter.Aggregation {
	// UNIQUE_COUNT aggregation requires string property value
	case models.MeterAggregationUniqueCount:
		switch valueRaw.(type) {
		case string, float64:
			// No need to do anything
		default:
			return NewProcessingError("event data value property must be string for unique count aggregation", INVALID)
		}
	// SUM, AVG, MIN, MAX aggregations require float64 parsable value property value
	case models.MeterAggregationSum, models.MeterAggregationAvg, models.MeterAggregationMin, models.MeterAggregationMax:
		switch value := valueRaw.(type) {
		case string:
			_, err = strconv.ParseFloat(value, 64)
			if err != nil {
				return NewProcessingError(fmt.Sprintf("event data value cannot be parsed as float64: %s", value), INVALID)
			}
		case float64:
			// No need to do anything
		default:
			return NewProcessingError("event data value property cannot be parsed", INVALID)
		}
	}

	return nil
}
