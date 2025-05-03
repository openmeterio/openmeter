package svix

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

func (h svixHandler) SendMessage(ctx context.Context, params webhook.SendMessageInput) (*webhook.Message, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate SendMessageInputs: %w", err)
	}

	var eventID *string
	if params.EventID != "" {
		eventID = &params.EventID
	}

	input := svix.MessageIn{
		Channels:  params.Channels,
		EventId:   eventID,
		EventType: params.EventType,
		Payload:   params.Payload,
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	o, err := h.client.Message.Create(ctx, params.Namespace, input, &svix.MessageCreateOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		err = unwrapSvixError(err)

		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &webhook.Message{
		Namespace: params.Namespace,
		ID:        o.Id,
		EventID:   lo.FromPtr(o.EventId),
		EventType: o.EventType,
		Channels:  o.Channels,
		Payload:   o.Payload,
	}, nil
}
