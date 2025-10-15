package svix

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/idempotency"
)

func (h svixHandler) SendMessage(ctx context.Context, params webhook.SendMessageInput) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid send message params: %w", err)
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

		idempotencyKey, err := idempotency.Key()
		if err != nil {
			return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.event_id", lo.FromPtr(input.EventId)),
			attribute.String("svix.event_type", input.EventType),
			attribute.String("svix.app_id", params.Namespace),
			attribute.String("idempotency_key", idempotencyKey),
		}

		span.AddEvent("sending message", trace.WithAttributes(spanAttrs...))

		o, err := h.client.Message.Create(ctx, params.Namespace, input, &svix.MessageCreateOptions{
			IdempotencyKey: &idempotencyKey,
			WithContent:    lo.ToPtr(true),
		})
		if err = internal.WrapSvixError(err); err != nil {
			var svixErr Error

			span.RecordError(err, trace.WithAttributes(spanAttrs...))

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

	return tracex.Start[*webhook.Message](ctx, h.tracer, "svix.send_message").Wrap(fn)
}

func (h svixHandler) getMessage(ctx context.Context, namespace, eventID string) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		o, err := h.client.Message.Get(ctx, namespace, eventID, &svix.MessageGetOptions{
			WithContent: lo.ToPtr(true),
		})
		if err = internal.WrapSvixError(err); err != nil {
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

	return tracex.Start[*webhook.Message](ctx, h.tracer, "svix.get_message").Wrap(fn)
}
