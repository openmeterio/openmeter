package svix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
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
	if err = internal.AsSvixError(err); err != nil {
		var svixErr Error

		ok := errors.As(err, &svixErr)
		if !ok {
			return nil, fmt.Errorf("failed to send message: %w", err)
		}

		switch svixErr.HTTPStatus {
		case internal.HTTPStatusValidationError:
			return nil, webhook.NewValidationError(svixErr)
		case http.StatusConflict:
			return h.getMessage(ctx, params.Namespace, params.EventID)
		default:
			return nil, fmt.Errorf("failed to send message: %w", svixErr)
		}
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

func (h svixHandler) getMessage(ctx context.Context, namespace, eventID string) (*webhook.Message, error) {
	o, err := h.client.Message.Get(ctx, namespace, eventID, &svix.MessageGetOptions{
		WithContent: lo.ToPtr(true),
	})
	if err = internal.AsSvixError(err); err != nil {

		var svixErr Error

		ok := errors.As(err, &svixErr)
		if !ok {
			return nil, fmt.Errorf("failed to send message: %w", err)
		}

		if svixErr.HTTPStatus == internal.HTTPStatusValidationError {
			return nil, webhook.NewValidationError(svixErr)
		}

		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &webhook.Message{
		Namespace: namespace,
		ID:        o.Id,
		EventID:   lo.FromPtr(o.EventId),
		EventType: o.EventType,
		Channels:  o.Channels,
		Payload:   o.Payload,
	}, nil
}
