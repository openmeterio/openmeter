package sink

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type MeterStore struct {
	Meters []meterpkg.Meter
}

type NamespaceStore struct {
	namespaces map[string]*MeterStore
}

func NewNamespaceStore() *NamespaceStore {
	return &NamespaceStore{
		namespaces: make(map[string]*MeterStore),
	}
}

func (n *NamespaceStore) AddMeter(meter meterpkg.Meter) {
	if n.namespaces[meter.Namespace] == nil {
		n.namespaces[meter.Namespace] = &MeterStore{
			Meters: []meterpkg.Meter{meter},
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
			if m.Status.Error != nil {
				return
			}

			// Parse kafka event
			event, err := serializer.FromKafkaPayloadToCloudEvents(*m.Serialized)
			if err != nil {
				m.Status = sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: errors.New("cannot unmarshal event data"),
				}

				return
			}

			// Parse event with meter
			_, _, _, err = meterpkg.ParseEvent(meter, event)
			if err != nil {
				m.Status = sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: err,
				}

				return
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
