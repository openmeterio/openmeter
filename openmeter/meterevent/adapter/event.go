package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

func (a *adapter) ListEvents(ctx context.Context, input meterevent.ListEventsInput) ([]api.IngestedEvent, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	// Get all events
	events, err := a.streamingConnector.ListEvents(ctx, input.Namespace, streaming.ListEventsParams{
		ClientID:       input.ClientID,
		From:           input.From,
		To:             input.To,
		IngestedAtFrom: input.IngestedAtFrom,
		IngestedAtTo:   input.IngestedAtTo,
		ID:             input.ID,
		Subject:        input.Subject,
		Limit:          input.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	// Get all meters
	meters, err := meter.ListAll(ctx, a.meterService, meter.ListMetersParams{
		Namespace: input.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("get meters: %w", err)
	}

	// Validate events against meters
	for idx, event := range events {
		for _, m := range meters {
			if event.Event.Type() == m.EventType {
				_, _, _, err := meter.ParseEvent(m, event.Event)
				if err != nil {
					events[idx].ValidationError = lo.ToPtr(err.Error())
				}
			}
		}
	}

	return events, nil
}
