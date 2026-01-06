package svix

import (
	"context"
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
			attribute.String(AnnotationMessageEventID, lo.FromPtr(input.EventId)),
			attribute.String(AnnotationEventType, input.EventType),
			attribute.String(AnnotationApplicationUID, params.Namespace),
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
				// Return a custom error to indicate that the message is not ready yet.
				if svixErr.HTTPStatus == http.StatusConflict {
					return nil, webhook.NewMessageAlreadyExistsError(params.Namespace, params.EventID)
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
			Payload:          &params.Payload,
			DeliveryStatuses: nil,
			Timestamp:        out.Timestamp,
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

func (h svixHandler) getDeliveryStatus(ctx context.Context, namespace, eventID, channelID string) ([]webhook.MessageDeliveryStatus, error) {
	fn := func(ctx context.Context) ([]webhook.MessageDeliveryStatus, error) {
		if namespace == "" || eventID == "" {
			return nil, fmt.Errorf("invalid params")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationMessageEventID, eventID),
			attribute.String(AnnotationApplicationUID, namespace),
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
					return nil, fmt.Errorf("failed to map delivery attempt state [svix.message.status=%d svix.message.status_text=%s]",
						attempt.Status, attempt.StatusText)
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
			attribute.Int(AnnotationMessageAttemptsCount, len(attemptsBySvixEndpointID)),
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
					return nil, fmt.Errorf("failed to map delivery status [svix.message.status=%d svix.message.status_text=%s]",
						dest.Status, dest.StatusText)
				}

				if channelID != "" && !lo.Contains(dest.Channels, channelID) {
					continue
				}

				if dest.Uid == nil {
					h.logger.WarnContext(ctx, "ignoring webhook endpoint with no uid set",
						"svix.app", namespace,
						"svix.event.id", eventID,
						"svix.endpoint.id", dest.Id,
					)

					continue
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
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get message params: %w", err)
		}

		msgID := lo.CoalesceOrEmpty(params.ID, params.EventID)

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationMessageID, params.ID),
			attribute.String(AnnotationMessageEventID, params.EventID),
			attribute.String(AnnotationApplicationUID, params.Namespace),
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

		if params.Expand.DeliveryStatusByChannelID != "" {
			span.AddEvent("fetching webhook message delivery statuses", trace.WithAttributes(spanAttrs...))

			statuses, err = h.getDeliveryStatus(ctx, params.Namespace, msgOut.Id, params.Expand.DeliveryStatusByChannelID)
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

func (h svixHandler) ResendMessage(ctx context.Context, params webhook.ResendMessageInput) error {
	fn := func(ctx context.Context) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("invalid resend message params: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationMessageID, params.ID),
			attribute.String(AnnotationMessageEventID, params.EventID),
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, params.ChannelID),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("resend webhook message", trace.WithAttributes(spanAttrs...))

		idempotencyKey, err := idempotency.Key()
		if err != nil {
			return fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		_, err = h.client.MessageAttempt.Resend(ctx, params.Namespace, params.EventID, params.ChannelID, &svix.MessageAttemptResendOptions{
			IdempotencyKey: &idempotencyKey,
		})
		if err = internal.WrapSvixError(err); err != nil {
			return fmt.Errorf("failed to resend message [svix.app=%s svix.message.id=%s svix.endpoint.uid=%s]: %w",
				params.Namespace, params.EventID, params.ChannelID, err)
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "svix.resend_message").Wrap(fn)
}

func deliveryStateFromSvixMessageStatus(status svixmodels.MessageStatus) notification.EventDeliveryStatusState {
	switch status {
	case svixmodels.MESSAGESTATUS_SUCCESS:
		return notification.EventDeliveryStatusStateSuccess
	case svixmodels.MESSAGESTATUS_PENDING, svixmodels.MESSAGESTATUS_SENDING:
		return notification.EventDeliveryStatusStateSending
	case svixmodels.MESSAGESTATUS_FAIL:
		return notification.EventDeliveryStatusStateFailed
	default:
		return notification.EventDeliveryStatusState(fmt.Sprintf("unknown_status: %d", status))
	}
}
