package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) ListEvents(ctx context.Context, params meterevent.ListEventsParams) ([]meterevent.Event, error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	// Get all events
	events, err := a.streamingConnector.ListEvents(ctx, params.Namespace, params)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	// Get all meters
	meters, err := meter.ListAll(ctx, a.meterService, meter.ListMetersParams{
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("get meters: %w", err)
	}

	// Validate events against meters
	validatedEvents := make([]meterevent.Event, len(events))
	for idx, event := range events {
		validatedEvent := meterevent.Event{
			ID:               event.ID,
			Type:             event.Type,
			Source:           event.Source,
			Subject:          event.Subject,
			Time:             event.Time,
			Data:             event.Data,
			IngestedAt:       event.IngestedAt,
			StoredAt:         event.StoredAt,
			ValidationErrors: make([]error, 0),
		}

		for _, m := range meters {
			if event.Type == m.EventType {
				_, _, _, err := meter.ParseEvent(m, event.Data)
				if err != nil {
					validatedEvent.ValidationErrors = append(validatedEvent.ValidationErrors, err)
				}
			}
		}

		validatedEvents[idx] = validatedEvent
	}

	return validatedEvents, nil
}
