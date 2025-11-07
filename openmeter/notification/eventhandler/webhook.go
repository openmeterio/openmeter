package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

var (
	ErrUserSendAttemptsExhausted       = errors.New("user error: multiple failed to send event due to errors from the recipient. retry will NOT be attempted")
	ErrSystemDispatchAttemptsExhausted = errors.New("system error: multiple dispatch attempts failed. retry will NOT be attempted")
	ErrSystemRecoverableError          = errors.New("system error: unexpected error happened during sending event. retry will be attempted")
	ErrSystemUnrecoverableError        = errors.New("system error: unrecoverable error happened during sending event. retry will NOT be attempted")
)

func (h *Handler) reconcileWebhookEvent(ctx context.Context, event *notification.Event) error {
	fn := func(ctx context.Context) error {
		if event == nil {
			return fmt.Errorf("event must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.namespace", event.Namespace),
			attribute.Stringer("notification.event.type", event.Type),
			attribute.String("notification.event.id", event.ID),
		}

		span.SetAttributes(spanAttrs...)

		h.logger.DebugContext(ctx, "reconciling webhook event", "event", event)

		// Sort the delivery statuses with webhook channel type by priority and filter out non-active statuses (success, failed).
		sortedActiveStatuses := sortDeliveryStatusStateByPriority(
			filterActiveDeliveryStatusesByChannelType(event, notification.ChannelTypeWebhook),
		)

		// If there are no active statuses, nothing to reconcile so return.
		if len(sortedActiveStatuses) == 0 {
			return nil
		}

		// Check if the message already exists in the webhook provider.
		// Return the error if it is other than not found as it is going to be handled by the reconciler logic.
		msg, err := h.getWebhookMessage(ctx, event)
		if err != nil && !webhook.IsNotFoundError(err) {
			return fmt.Errorf("failed to get webhook message: %w", err)
		}

		var errs []error

		for _, status := range sortedActiveStatuses {
			// Skip the delivery status update if next_attempt is set, and it is in the future.
			if next := lo.FromPtr(status.NextAttempt); !next.IsZero() && next.After(clock.Now()) {
				span.AddEvent("skipping delivery status update due to no next_attempt is set or in the future", trace.WithAttributes(spanAttrs...),
					trace.WithAttributes(attribute.String("next_attempt", next.UTC().Format(time.RFC3339))),
				)

				continue
			}

			h.logger.DebugContext(ctx, "reconciling delivery status", "delivery_status.state", status, "namespace", event.Namespace, "event_id", event.ID)

			var input *notification.UpdateEventDeliveryStatusInput

			switch status.State {
			case notification.EventDeliveryStatusStatePending:
				// Check if the delivery status is pending for too long.
				nextAttempt := lo.FromPtr(status.NextAttempt)
				if nextAttempt.IsZero() {
					nextAttempt = clock.Now()
				}

				if nextAttempt.Sub(status.CreatedAt) > h.pendingTimeout {
					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        notification.EventDeliveryStatusStateFailed,
						Reason:       ErrSystemDispatchAttemptsExhausted.Error(),
						Annotations:  status.Annotations,
						NextAttempt:  nil,
						Attempts:     status.Attempts,
					}

					break
				}

				switch {
				case msg != nil:
					span.AddEvent("webhook message fetched from provider", trace.WithAttributes(spanAttrs...))

					attempts := status.Attempts
					state := notification.EventDeliveryStatusStateSending
					nextAttempt := lo.ToPtr(clock.Now().Add(h.reconcileInterval))

					msgStatusByChannel := getDeliveryStatusByChannelID(lo.FromPtr(msg.DeliveryStatuses), status.ChannelID)

					if msgStatusByChannel != nil {
						attempts = msgStatusByChannel.Attempts
						state = msgStatusByChannel.State
						nextAttempt = msgStatusByChannel.NextAttempt
					}

					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        state,
						Annotations:  status.Annotations,
						NextAttempt:  nextAttempt,
						Attempts:     attempts,
					}
				case webhook.IsNotFoundError(err):
					// Event is not yet sent to webhook provider.
					msg, err = h.sendWebhookMessage(ctx, event)

					switch {
					case webhook.IsMessageNotReadyError(err):
						// Event is sent to the provider but has not been processed yet.
						// Keep it in pending state and update the next attempt.

						span.AddEvent("webhook message is already sent to provider but it has not been processed", trace.WithAttributes(spanAttrs...))

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStatePending,
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}
					case webhook.IsUnrecoverableError(err):
						// Unrecoverable error happened

						span.AddEvent("fetching webhook message from provider returned unrecoverable error", trace.WithAttributes(spanAttrs...))
						span.RecordError(err, trace.WithAttributes(spanAttrs...))

						h.logger.ErrorContext(ctx, "fetching webhook message from provider returned unrecoverable error",
							"error", err.Error(), "delivery_status.state", status.State, "namespace", event.Namespace, "event_id", event.ID)

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       ErrSystemUnrecoverableError.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}
					case err != nil:
						// Transient error happened

						span.AddEvent("fetching webhook message from provider returned transient error", trace.WithAttributes(spanAttrs...))
						span.RecordError(err, trace.WithAttributes(spanAttrs...))

						h.logger.WarnContext(ctx, "fetching webhook message from provider returned transient error",
							"error", err.Error(), "delivery_status.state", status.State, "namespace", event.Namespace, "event_id", event.ID)

						retryAfter := h.reconcileInterval

						rErr, ok := lo.ErrorsAs[webhook.RetryableError](err)
						if ok {
							retryAfter = rErr.RetryAfter()
						}

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStatePending,
							Reason:       ErrSystemRecoverableError.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  lo.ToPtr(clock.Now().Add(retryAfter)),
							Attempts:     status.Attempts,
						}
					case msg != nil:
						span.AddEvent("webhook message is sent to provider", trace.WithAttributes(spanAttrs...))

						attempts := status.Attempts
						state := notification.EventDeliveryStatusStateSending
						nextAttempt := lo.ToPtr(clock.Now().Add(h.reconcileInterval))

						msgStatusByChannel := getDeliveryStatusByChannelID(lo.FromPtr(msg.DeliveryStatuses), status.ChannelID)

						if msgStatusByChannel != nil {
							attempts = msgStatusByChannel.Attempts
							state = msgStatusByChannel.State
							nextAttempt = msgStatusByChannel.NextAttempt
						}

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        state,
							Annotations:  status.Annotations,
							NextAttempt:  nextAttempt,
							Attempts:     attempts,
						}
					default:
						span.RecordError(err, trace.WithAttributes(spanAttrs...))
						errs = append(errs, fmt.Errorf("unhandled reconciling state: %s", status.State.String()))
					}
				default:
					span.RecordError(err, trace.WithAttributes(spanAttrs...))
					errs = append(errs, fmt.Errorf("unhandled reconciling state: %s", status.State.String()))
				}
			case notification.EventDeliveryStatusStateSending:
				// Check if the delivery status is sending for too long.
				nextAttempt := lo.FromPtr(status.NextAttempt)
				if nextAttempt.IsZero() {
					nextAttempt = clock.Now()
				}

				if nextAttempt.Sub(status.CreatedAt) > h.sendingTimeout {
					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        notification.EventDeliveryStatusStateFailed,
						Reason:       ErrUserSendAttemptsExhausted.Error(),
						Annotations:  status.Annotations,
						NextAttempt:  nil,
						Attempts:     status.Attempts,
					}

					break
				}

				switch {
				case webhook.IsNotFoundError(err):
					span.RecordError(err, trace.WithAttributes(spanAttrs...))

					stuckInSendingSince := status.UpdatedAt
					if status.NextAttempt != nil && !status.NextAttempt.IsZero() {
						stuckInSendingSince = *status.NextAttempt
					}

					if clock.Now().Sub(stuckInSendingSince) > time.Hour {
						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStatePending,
							Annotations:  status.Annotations,
							NextAttempt:  status.NextAttempt,
							Attempts:     status.Attempts,
						}
					}
				case msg != nil:
					span.AddEvent("fetched webhook message from provider", trace.WithAttributes(spanAttrs...))

					msgStatusByChannel := getDeliveryStatusByChannelID(lo.FromPtr(msg.DeliveryStatuses), status.ChannelID)

					if msgStatusByChannel == nil {
						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       ErrSystemUnrecoverableError.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}

						break
					}

					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        msgStatusByChannel.State,
						Annotations:  status.Annotations.Merge(msg.Annotations),
						NextAttempt:  msgStatusByChannel.NextAttempt,
						Attempts:     msgStatusByChannel.Attempts,
					}
				default:
					span.RecordError(err, trace.WithAttributes(spanAttrs...))
					errs = append(errs, fmt.Errorf("unhandled reconciling state: %s", status.State.String()))
				}
			case notification.EventDeliveryStatusStateSuccess, notification.EventDeliveryStatusStateFailed:
				h.logger.DebugContext(ctx, "reconciling delivery status state", "delivery_status.state", status.State, "namespace", event.Namespace, "event_id", event.ID)
			default:
				errs = append(errs, fmt.Errorf("unsupported delivery status state: %s", status.State.String()))
			}

			if input == nil {
				span.AddEvent("no update for delivery status", trace.WithAttributes(spanAttrs...))
				h.logger.DebugContext(ctx, "no update for delivery status")

				continue
			}

			_, err = h.repo.UpdateEventDeliveryStatus(ctx, *input)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to update event delivery [namespace=%s notification.delivery_status.id=%s]: %w", status.Namespace, status.ID, err))
			}
		}

		err = errors.Join(errs...)
		if err != nil {
			h.logger.ErrorContext(ctx, "reconciling webhook event has errors", "errors", errs)
		}

		return err
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile_event.webhook").Wrap(fn)
}

func getDeliveryStatusByChannelID(items []webhook.MessageDeliveryStatus, channelID string) *webhook.MessageDeliveryStatus {
	for _, item := range items {
		if item.ChannelID == channelID {
			return &item
		}
	}

	return nil
}

func (h *Handler) getOrSendWebhookMessage(ctx context.Context, event *notification.Event) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		if event == nil {
			return nil, fmt.Errorf("event must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.namespace", event.Namespace),
			attribute.Stringer("notification.event.type", event.Type),
			attribute.String("notification.event.id", event.ID),
		}

		span.SetAttributes(spanAttrs...)

		msg, err := h.getWebhookMessage(ctx, event)
		if err != nil || webhook.IsNotFoundError(err) {
			return nil, fmt.Errorf("failed to get webhook message: %w", err)
		}

		if msg != nil {
			return msg, nil
		}

		msg, err = h.sendWebhookMessage(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to send webhook message: %w", err)
		}

		return msg, nil
	}

	return tracex.Start[*webhook.Message](ctx, h.tracer, "event_handler.get_or_webhook_message").Wrap(fn)
}

func (h *Handler) getWebhookMessage(ctx context.Context, event *notification.Event) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		if event == nil {
			return nil, fmt.Errorf("event must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.namespace", event.Namespace),
			attribute.Stringer("notification.event.type", event.Type),
			attribute.String("notification.event.id", event.ID),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("getting webhook message", trace.WithAttributes(spanAttrs...))

		msg, err := h.webhook.GetMessage(ctx, webhook.GetMessageInput{
			Namespace: event.Namespace,
			EventID:   event.ID,
			Expand: webhook.ExpandParams{
				DeliveryStatus: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get message for webhook channel: %w", err)
		}

		return msg, nil
	}

	return tracex.Start[*webhook.Message](ctx, h.tracer, "event_handler.get_webhook_message").Wrap(fn)
}

func (h *Handler) sendWebhookMessage(ctx context.Context, event *notification.Event) (*webhook.Message, error) {
	fn := func(ctx context.Context) (*webhook.Message, error) {
		if event == nil {
			return nil, fmt.Errorf("event must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.namespace", event.Namespace),
			attribute.Stringer("notification.event.type", event.Type),
			attribute.String("notification.event.id", event.ID),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("sending webhook message", trace.WithAttributes(spanAttrs...))

		msg, err := h.webhook.GetMessage(ctx, webhook.GetMessageInput{
			Namespace: event.Namespace,
			EventID:   event.ID,
			Expand: webhook.ExpandParams{
				DeliveryStatus: true,
			},
		})
		if err != nil && !webhook.IsNotFoundError(err) {
			return nil, fmt.Errorf("failed to get message for webhook channel: %w", err)
		}

		if msg != nil {
			return msg, nil
		}

		payload, err := eventAsPayload(event)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize webhook message payload: %w", err)
		}

		msg, err = h.webhook.SendMessage(ctx, webhook.SendMessageInput{
			Namespace: event.Namespace,
			EventID:   event.ID,
			EventType: event.Type.String(),
			Channels:  []string{event.Rule.ID},
			Payload:   payload,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to send webhook message to webhook provider: %w", err)
		}

		return msg, nil
	}

	return tracex.Start[*webhook.Message](ctx, h.tracer, "event_handler.send_webhook_message").Wrap(fn)
}

func eventAsPayload(event *notification.Event) (webhook.Payload, error) {
	var (
		payload any
		err     error
	)

	switch event.Type {
	case notification.EventTypeBalanceThreshold:
		payload, err = httpdriver.FromEventAsBalanceThresholdPayload(*event)
		if err != nil {
			return nil, fmt.Errorf("failed to cast event payload: %w", err)
		}
	case notification.EventTypeEntitlementReset:
		payload, err = httpdriver.FromEventAsEntitlementResetPayload(*event)
		if err != nil {
			return nil, fmt.Errorf("failed to cast event payload: %w", err)
		}
	case notification.EventTypeInvoiceCreated:
		payload, err = httpdriver.FromEventAsInvoiceCreatedPayload(*event)
		if err != nil {
			return nil, fmt.Errorf("failed to cast event payload: %w", err)
		}
	case notification.EventTypeInvoiceUpdated:
		payload, err = httpdriver.FromEventAsInvoiceUpdatedPayload(*event)
		if err != nil {
			return nil, fmt.Errorf("failed to cast event payload: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}

	var m webhook.Payload

	m, err = notification.AsRawPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to cast event payload: %w", err)
	}

	return m, nil
}
