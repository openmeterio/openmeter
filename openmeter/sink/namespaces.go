package sink

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	ommeter "github.com/openmeterio/openmeter/openmeter/meter"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
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
			if m.Status.Error != nil {
				return
			}

			// Parse kafka event
			event, err := kafkaPayloadToCloudEvents(*m.Serialized)
			if err != nil {
				m.Status = sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: errors.New("cannot parse event"),
				}
			}

			// Validate event against meter
			value, groupBy, err := ommeter.ParseEvent(meter, event)
			if err != nil {
				m.Status = sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: err,
				}

				return
			}

			m.MeterEvents = append(m.MeterEvents, sinkmodels.MeterEvent{
				Meter:   &meter,
				Value:   value,
				GroupBy: groupBy,
			})
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

func kafkaPayloadToCloudEvents(payload serializer.CloudEventsKafkaPayload) (event.Event, error) {
	ev := event.New()

	ev.SetID(payload.Id)
	ev.SetType(payload.Type)
	ev.SetSource(payload.Source)
	ev.SetSubject(payload.Subject)
	ev.SetTime(payload.Time)

	err := ev.SetData(event.ApplicationJSON, []byte(payload.Data))
	if err != nil {
		return event.Event{}, err
	}

	return ev, nil
}
