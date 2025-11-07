package svix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"
	svixmodels "github.com/svix/svix-webhooks/go/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/idempotency"
	"github.com/openmeterio/openmeter/pkg/models"
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
			attribute.String("svix.event.id", lo.FromPtr(input.EventId)),
			attribute.String("svix.event.type", input.EventType),
			attribute.String("svix.app.id", params.Namespace),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("sending message", trace.WithAttributes(spanAttrs...), trace.WithAttributes(
			attribute.String("idempotency_key", idempotencyKey),
		))

		out, err := h.client.Message.Create(ctx, params.Namespace, input, &svix.MessageCreateOptions{
			IdempotencyKey: &idempotencyKey,
			WithContent:    lo.ToPtr(false),
		})
		if err = internal.WrapSvixError(err); err != nil {
			span.RecordError(err, trace.WithAttributes(spanAttrs...))

			if svixErr, ok := lo.ErrorsAs[Error](err); ok {
				// Conflict received in case the EventID already exist meaning the message is already published.
				// Let's try to get the message from the API.
				if svixErr.HTTPStatus == http.StatusConflict {
					return nil, webhook.NewMessageNotReadyError(params.Namespace, params.EventID)
				}

				return nil, fmt.Errorf("failed to send message: %w", err)
			}

			return nil, fmt.Errorf("failed to send message: %w", err)
		}

		return &webhook.Message{
			Namespace:        params.Namespace,
			ID:               out.Id,
			EventID:          lo.FromPtr(out.EventId),
			EventType:        out.EventType,
			Channels:         out.Channels,
			Payload:          &out.Payload,
			DeliveryStatuses: nil,
			Annotations: models.Annotations{
				AnnotationMessageID:      out.Id,
				AnnotationMessageEventID: out.EventId,
				AnnotationEventType:      out.EventType,
				AnnotationApplicationUID: params.Namespace,
			},
		}, nil
	}

	return tracex.Start[*webhook.Message](ctx, h.tracer, "svix.send_message").Wrap(fn)
}

const (
	ListAttemptLimit uint64 = 25
)

func (h svixHandler) getDeliveryStatus(ctx context.Context, namespace, eventID string) ([]webhook.MessageDeliveryStatus, error) {
	fn := func(ctx context.Context) ([]webhook.MessageDeliveryStatus, error) {
		if namespace == "" || eventID == "" {
			return nil, fmt.Errorf("invalid params")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.event.id", eventID),
			attribute.String("svix.app.id", namespace),
		}

		span.SetAttributes(spanAttrs...)

		// Fetch delivery attempts for the event.

		span.AddEvent("fetching delivery attempts", trace.WithAttributes(spanAttrs...))

		var attemptsIterator *string

		attemptsBySvixEndpointID := make(map[string][]notification.EventDeliveryAttempt)

		for {
			attemptsByMsgOut, err := h.client.MessageAttempt.ListByMsg(ctx, namespace, eventID, &svix.MessageAttemptListByMsgOptions{
				Limit:    lo.ToPtr[uint64](ListAttemptLimit),
				Iterator: attemptsIterator,
			})
			if err = internal.WrapSvixError(err); err != nil {
				return nil, fmt.Errorf("failed to get webhook message attempts: %w", err)
			}

			for _, attempt := range attemptsByMsgOut.Data {
				state := deliveryStateFromSvixMessageStatus(attempt.Status)

				if state == "" {
					return nil, fmt.Errorf("failed to map delivery attempt state [svix.message.status=%d svix.message.status_test=%s]: %w",
						attempt.Status, attempt.StatusText, err)
				}

				deliveryAttempt := notification.EventDeliveryAttempt{
					State: state,
					Response: notification.EventDeliveryAttemptResponse{
						StatusCode: lo.ToPtr(int(attempt.ResponseStatusCode)),
						Body:       attempt.Response,
						Duration:   time.Duration(attempt.ResponseDurationMs) * time.Millisecond,
						URL:        lo.ToPtr(attempt.Url),
					},
					Timestamp: attempt.Timestamp,
				}

				attempts, ok := attemptsBySvixEndpointID[attempt.EndpointId]
				if !ok {
					attempts = []notification.EventDeliveryAttempt{
						deliveryAttempt,
					}
				} else {
					attempts = append(attempts, deliveryAttempt)
				}

				attemptsBySvixEndpointID[attempt.EndpointId] = attempts
			}

			// Continue if there is more data to fetch.
			if !attemptsByMsgOut.Done {
				attemptsIterator = attemptsByMsgOut.Iterator

				// Wait before fetching the next page to avoid hitting the rate limit.
				time.Sleep(100 * time.Millisecond)

				continue
			}

			break
		}

		span.AddEvent("delivery attempts", trace.WithAttributes(spanAttrs...), trace.WithAttributes(
			attribute.Int("svix.message.attempts_count", len(attemptsBySvixEndpointID)),
		))

		// Fetch delivery attempts for the event by endpoints.

		span.AddEvent("fetching delivery latest delivery status by endpoints", trace.WithAttributes(spanAttrs...))

		var (
			endpointsIterator *string
			statuses          []webhook.MessageDeliveryStatus
		)

		for {
			endpointOut, err := h.client.MessageAttempt.ListAttemptedDestinations(ctx, namespace, eventID, &svix.MessageAttemptListAttemptedDestinationsOptions{
				Limit:    lo.ToPtr[uint64](ListAttemptLimit),
				Iterator: endpointsIterator,
			})
			if err = internal.WrapSvixError(err); err != nil {
				return nil, fmt.Errorf("failed to get webhook message attempts: %w", err)
			}

			for _, dest := range endpointOut.Data {
				state := deliveryStateFromSvixMessageStatus(dest.Status)

				// As long as the NextAttempt is not nil the delivery status is in a transient state.
				// Svix returns FAIL even if it is still attempting to send the message.
				if state == notification.EventDeliveryStatusStateFailed && dest.NextAttempt != nil {
					state = notification.EventDeliveryStatusStateSending
				}

				if state == "" {
					return nil, fmt.Errorf("failed to map delivery status [svix.message.status=%d svix.message.status_test=%s]: %w",
						dest.Status, dest.StatusText, err)
				}

				if dest.Uid == nil {
					return nil, fmt.Errorf("invalid webhook endpiont: uid is nil")
				}

				statuses = append(statuses, webhook.MessageDeliveryStatus{
					NextAttempt: dest.NextAttempt,
					State:       state,
					ChannelID:   lo.FromPtr(dest.Uid),
					Attempts:    attemptsBySvixEndpointID[dest.Id],
				})
			}

			// Continue if there is more data to fetch.
			if !endpointOut.Done {
				endpointsIterator = endpointOut.Iterator

				// Wait before fetching the next page to avoid hitting the rate limit.
				time.Sleep(100 * time.Millisecond)

				continue
			}

			break
		}

		return statuses, nil
	}

	return tracex.Start[[]webhook.MessageDeliveryStatus](ctx, h.tracer, "svix.get_delivery_status").Wrap(fn)
}

func (h svixHandler) GetMessage(ctx context.Context, params webhook.GetMessageInput) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		msgID := lo.CoalesceOrEmpty(params.ID, params.EventID)

		if msgID == "" {
			return nil, webhook.NewValidationError(
				errors.New("either svix message id or event id must be provided"),
			)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.event.id", params.ID),
			attribute.String("svix.app.id", params.Namespace),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("fetching webhook message", trace.WithAttributes(spanAttrs...))

		msgOut, err := h.client.Message.Get(ctx, params.Namespace, msgID, &svix.MessageGetOptions{
			WithContent: lo.ToPtr(params.Expand.Payload),
		})
		if err = internal.WrapSvixError(err); err != nil {
			return nil, fmt.Errorf("failed to get message: %w", err)
		}

		var statuses []webhook.MessageDeliveryStatus

		if params.Expand.DeliveryStatus {
			span.AddEvent("fetching webhook message delivery statuses", trace.WithAttributes(spanAttrs...))

			statuses, err = h.getDeliveryStatus(ctx, params.Namespace, msgOut.Id)
			if err != nil {
				return nil, fmt.Errorf("failed to get message delivery statuses: %w", err)
			}
		}

		return &webhook.Message{
			Namespace:        params.Namespace,
			ID:               msgOut.Id,
			EventID:          lo.FromPtr(msgOut.EventId),
			EventType:        msgOut.EventType,
			Channels:         msgOut.Channels,
			Payload:          &msgOut.Payload,
			DeliveryStatuses: lo.ToPtr(statuses),
			Annotations: models.Annotations{
				AnnotationMessageID:      msgOut.Id,
				AnnotationMessageEventID: msgOut.EventId,
				AnnotationEventType:      msgOut.EventType,
				AnnotationApplicationUID: params.Namespace,
			},
		}, nil
	}

	return tracex.Start[*webhook.Message](ctx, h.tracer, "svix.get_message").Wrap(fn)
}

func deliveryStateFromSvixMessageStatus(status svixmodels.MessageStatus) notification.EventDeliveryStatusState {
	switch status {
	case svixmodels.MESSAGESTATUS_SUCCESS:
		return notification.EventDeliveryStatusStateSuccess
	case svixmodels.MESSAGESTATUS_PENDING:
		return notification.EventDeliveryStatusStateSending
	case svixmodels.MESSAGESTATUS_FAIL:
		return notification.EventDeliveryStatusStateFailed
	case svixmodels.MESSAGESTATUS_SENDING:
		return notification.EventDeliveryStatusStateSending
	default:
		return ""
	}
}
