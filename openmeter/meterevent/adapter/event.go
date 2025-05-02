package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ListEvents returns a list of events.
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

	// Validate events
	validatedEvents, err := a.validateEvents(ctx, params.Namespace, events)
	if err != nil {
		return nil, fmt.Errorf("validate events: %w", err)
	}

	return validatedEvents, nil
}

// ListEventsV2 returns a list of events.
func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (pagination.Result[meterevent.Event], error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Event]{}, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	// Get all events v2
	events, err := a.streamingConnector.ListEventsV2(ctx, params)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("query events: %w", err)
	}

	// Validate events
	validatedEvents, err := a.validateEvents(ctx, params.Namespace, events)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("validate events: %w", err)
	}

	return pagination.NewResult(validatedEvents), nil
}

// validateEvents validates a list of raw events against a list of meters.
func (a *adapter) validateEvents(ctx context.Context, namespace string, events []streaming.RawEvent) ([]meterevent.Event, error) {
	// Get all meters
	meters, err := meter.ListAll(ctx, a.meterService, meter.ListMetersParams{
		Namespace: namespace,
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

		meterMatch := false

		for _, m := range meters {
			if event.Type == m.EventType {
				meterMatch = true

				_, err = meter.ParseEventString(m, event.Data)
				if err != nil {
					validatedEvent.ValidationErrors = append(validatedEvent.ValidationErrors, err)
				}
			}
		}

		if !meterMatch {
			validatedEvent.ValidationErrors = append(validatedEvent.ValidationErrors, fmt.Errorf("no meter found for event type: %s", event.Type))
		}

		validatedEvents[idx] = validatedEvent
	}

	return validatedEvents, nil
}
