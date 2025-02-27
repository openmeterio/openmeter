package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) ListEvents(ctx context.Context, input meterevent.ListEventsInput) ([]api.IngestedEvent, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	events, err := a.streamingConnector.ListEvents(ctx, input.Namespace, streaming.ListEventsParams{
		From:           &input.From,
		To:             input.To,
		IngestedAtFrom: input.IngestedAtFrom,
		IngestedAtTo:   input.IngestedAtTo,
		ID:             input.ID,
		Subject:        input.Subject,
		HasError:       input.HasError,
		Limit:          input.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	return events, nil
}
