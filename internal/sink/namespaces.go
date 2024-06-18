package sink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/oliveagle/jsonpath"

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
func (a *NamespaceStore) ValidateEvent(_ context.Context, m *SinkMessage) {
	namespaceStore := a.namespaces[m.Namespace]
	if namespaceStore == nil {
		// We drop events from unknown org
		m.Status = ProcessingStatus{
			State: DROP,
			Error: fmt.Errorf("namespace not found: %s", m.Namespace),
		}

		return
	}

	// Validate a single event against multiple meters
	var foundMeter bool
	for _, meter := range namespaceStore.Meters {
		if meter.EventType == m.Serialized.Type {
			foundMeter = true
			validateEventWithMeter(meter, m)
			if m.Status.Error != nil {
				return
			}
			// A single event can match multiple meters so we cannot break the loop early
		}
	}

	if !foundMeter {
		// Mark as invalid so we can show it to the user
		m.Status = ProcessingStatus{
			State: INVALID,
			Error: fmt.Errorf("no meter found for event type: %s", m.Serialized.Type),
		}
	}
}

// validateEventWithMeter validates a single event against a single meter
func validateEventWithMeter(meter models.Meter, m *SinkMessage) {
	// Parse CloudEvents data as JSON, currently we only support JSON encoding
	var data interface{}
	err := json.Unmarshal([]byte(m.Serialized.Data), &data)
	if err != nil {
		m.Status = ProcessingStatus{
			State: INVALID,
			Error: errors.New("cannot unmarshal event data as json"),
		}

		return
	}

	// We can skip count events as they don't have value property
	if meter.Aggregation == models.MeterAggregationCount {
		return
	}

	// Get value from event data by value property
	valueRaw, err := jsonpath.JsonPathLookup(data, meter.ValueProperty)
	if err != nil {
		m.Status = ProcessingStatus{
			State: INVALID,
			Error: fmt.Errorf("event data is missing value property at %s", meter.ValueProperty),
		}

		return
	}
	if valueRaw == nil {
		m.Status = ProcessingStatus{
			State: INVALID,
			Error: errors.New("event data value cannot be null"),
		}

		return
	}

	// Aggregation specific value validation
	switch meter.Aggregation {
	// UNIQUE_COUNT aggregation requires string property value
	case models.MeterAggregationUniqueCount:
		switch valueRaw.(type) {
		case string, float64:
			// No need to do anything
		default:
			m.Status = ProcessingStatus{
				State: INVALID,
				Error: errors.New("event data value property must be string for unique count aggregation"),
			}

			return
		}
	// SUM, AVG, MIN, MAX aggregations require float64 parsable value property value
	case models.MeterAggregationSum, models.MeterAggregationAvg, models.MeterAggregationMin, models.MeterAggregationMax:
		switch value := valueRaw.(type) {
		case string:
			_, err = strconv.ParseFloat(value, 64)
			if err != nil {
				m.Status = ProcessingStatus{
					State: INVALID,
					Error: fmt.Errorf("event data value cannot be parsed as float64: %s", value),
				}

				return
			}
		case float64:
			// No need to do anything
		default:
			m.Status = ProcessingStatus{
				State: INVALID,
				Error: errors.New("event data value property cannot be parsed"),
			}

			return
		}
	}
}
