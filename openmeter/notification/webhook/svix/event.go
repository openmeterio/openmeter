package svix

import (
	"context"
	"fmt"

	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

func (h svixHandler) RegisterEventTypes(ctx context.Context, params webhook.RegisterEventTypesInputs) error {
	for _, eventType := range params.EventTypes {
		input := svix.EventTypeUpdate{
			Description: eventType.Description,
			FeatureFlag: nil,
			GroupName:   &eventType.GroupName,
			Schemas:     &eventType.Schemas,
			Deprecated:  &eventType.Deprecated,
		}

		_, err := h.client.EventType.Update(ctx, eventType.Name, input)
		if err != nil {
			err = unwrapSvixError(err)

			return fmt.Errorf("failed to create event type: %w", err)
		}
	}

	return nil
}
