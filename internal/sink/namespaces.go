package sink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/oliveagle/jsonpath"

	sinkmodels "github.com/openmeterio/openmeter/internal/sink/models"
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
func (n *NamespaceStore) ValidateEvent(_ context.Context, m *sinkmodels.SinkMessage) {
	namespaceStore, ok := n.namespaces[m.Namespace]
	if !ok || namespaceStore == nil {
		// We drop events from unknown org
		m.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.DROP,
			Error: fmt.Errorf("namespace not found: %s", m.Namespace),
		}

		return
	}

	// Collect all meters associated with the event and validate event against them
	for _, meter := range namespaceStore.Meters {
		if meter.EventType == m.Serialized.Type {
			m.Meters = append(m.Meters, meter)
			// Validating the event until the first error, as the meter becomes invalid
			// afterwards, we don't need to validate the event against the rest.
			//
			// On the other hand we still want to collect the list of affected meters
			// for the FlushEventHandler.
			if m.Status.Error == nil {
				validateEventWithMeter(meter, m)
			}
		}
	}

	if len(m.Meters) == 0 {
		// Mark as invalid so we can show it to the user
		m.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.INVALID,
			Error: fmt.Errorf("no meter found for event type: %s", m.Serialized.Type),
		}
	}
}

// validateEventWithMeter validates a single event against a single meter
func validateEventWithMeter(meter models.Meter, m *sinkmodels.SinkMessage) {
	// Parse CloudEvents data as JSON, currently we only support JSON encoding
	var data interface{}
	err := json.Unmarshal([]byte(m.Serialized.Data), &data)
	if err != nil {
		m.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.INVALID,
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
		m.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.INVALID,
			Error: fmt.Errorf("event data is missing value property at %s", meter.ValueProperty),
		}

		return
	}
	if valueRaw == nil {
		m.Status = sinkmodels.ProcessingStatus{
			State: sinkmodels.INVALID,
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
			m.Status = sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
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
				m.Status = sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: fmt.Errorf("event data value cannot be parsed as float64: %s", value),
				}

				return
			}
		case float64:
			// No need to do anything
		default:
			m.Status = sinkmodels.ProcessingStatus{
				State: sinkmodels.INVALID,
				Error: errors.New("event data value property cannot be parsed"),
			}

			return
		}
	}
}
