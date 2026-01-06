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
	ErrSystemChannelDisabled           = errors.New("system error: channel is disabled. retry will NOT be attempted")
	ErrSystemChannelNotFound           = errors.New("system error: channel not found. retry will NOT be attempted")
	ErrSystemRuleNotAssignToChannel    = errors.New("system error: rule not assigned to channel. retry will NOT be attempted")
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
		var err error

		// Fetch the list of webhook endpoints for the active delivery statuses.
		var webhooksOut []webhook.Webhook

		webhooksOut, err = h.webhook.ListWebhooks(ctx, webhook.ListWebhooksInput{
			Namespace: event.Namespace,
			IDs: lo.Map(sortedActiveStatuses, func(item notification.EventDeliveryStatus, _ int) string {
				return item.ChannelID
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to list webhooks: %w", err)
		}

		// Map the webhook endpoints by channel ID, so it is easier to look up the endpoint for delivery status by a given channel ID.
		webhooksByChannelID := lo.SliceToMap(webhooksOut, func(item webhook.Webhook) (string, webhook.Webhook) {
			return item.ID, item
		})

		// Check if the message already exists in the webhook provider.
		// Return the error if it is other than not found as it is going to be handled by the reconciler logic.
		var msg *webhook.Message

		msg, err = h.getWebhookMessage(ctx, event)
		if err != nil && !webhook.IsNotFoundError(err) {
			return fmt.Errorf("failed to get webhook message: %w", err)
		}

		var errs []error

		now := clock.Now()

		for _, status := range sortedActiveStatuses {
			// Skip the delivery status update if next_attempt is set, and it is in the future.
			if next := lo.FromPtr(status.NextAttempt); !next.IsZero() && next.After(now) {
				span.AddEvent("skipping delivery status update: next_attempt is set in the future", trace.WithAttributes(spanAttrs...),
					trace.WithAttributes(attribute.String("next_attempt", next.UTC().Format(time.RFC3339))),
				)

				continue
			}

			h.logger.DebugContext(ctx, "reconciling delivery status",
				"notification.delivery_status.state", status,
				"namespace", event.Namespace,
				"notification.event.id", event.ID,
			)

			deliveryStatusAttrs := []attribute.KeyValue{
				attribute.Stringer("notification.delivery_status.state", status.State),
				attribute.String("notification.delivery_status.id", status.ID),
			}

			var input *notification.UpdateEventDeliveryStatusInput

			switch status.State {
			case notification.EventDeliveryStatusStatePending:
				// Check if the delivery status is pending for too long.
				if now.Sub(status.CreatedAt) > h.pendingTimeout {
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

				// The provider returned a not found error, which means that either the event has not been dispatched to the provider
				// or the provider has not processed the event yet. The latter case can be verified by resending the message and checking
				// if the message already exists error is returned indicating that the message has been dispatched.
				if webhook.IsNotFoundError(err) {
					msg, err = h.sendWebhookMessage(ctx, event)
				}

				switch {
				case webhook.IsMessageAlreadyExistsError(err):
					// Event is sent to the provider but has not been processed yet. Keep it in pending state and update the next attempt.

					span.AddEvent("webhook message is already sent to provider but it has not been processed",
						trace.WithAttributes(spanAttrs...),
						trace.WithAttributes(deliveryStatusAttrs...),
					)

					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        notification.EventDeliveryStatusStatePending,
						Annotations:  status.Annotations,
						NextAttempt:  nil,
						Attempts:     status.Attempts,
					}
				case webhook.IsUnrecoverableError(err), webhook.IsValidationError(err):
					// Unrecoverable error happened, no retry is possible.

					span.AddEvent("fetching webhook message from provider returned unrecoverable error",
						trace.WithAttributes(spanAttrs...),
						trace.WithAttributes(deliveryStatusAttrs...),
					)

					span.RecordError(err, trace.WithAttributes(spanAttrs...), trace.WithAttributes(deliveryStatusAttrs...))

					h.logger.ErrorContext(ctx, "fetching webhook message from provider returned unrecoverable error",
						"error", err.Error(),
						"notification.delivery_status.state", status.State,
						"namespace", event.Namespace,
						"notification.event.id", event.ID)

					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        notification.EventDeliveryStatusStateFailed,
						Reason:       ErrSystemUnrecoverableError.Error(),
						Annotations:  status.Annotations,
						NextAttempt:  nil,
						Attempts:     status.Attempts,
					}
				case err != nil:
					// Transient error happened, retry after a short delay.

					span.AddEvent("fetching webhook message from provider returned transient error",
						trace.WithAttributes(spanAttrs...),
						trace.WithAttributes(deliveryStatusAttrs...),
					)

					span.RecordError(err, trace.WithAttributes(spanAttrs...), trace.WithAttributes(deliveryStatusAttrs...))

					h.logger.WarnContext(ctx, "fetching webhook message from provider returned transient error",
						"error", err.Error(),
						"notification.delivery_status.state", status.State,
						"namespace", event.Namespace,
						"notification.event.id", event.ID,
					)

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
						NextAttempt:  lo.ToPtr(now.Add(retryAfter)),
						Attempts:     status.Attempts,
					}
				case msg != nil:
					// Event fetched from the provider successfully, however, the event delivery states might be missing in case
					// the provider has not populated the delivery statuses mostly because the event has not been processed yet.

					span.AddEvent("webhook message fetched from provider",
						trace.WithAttributes(spanAttrs...),
						trace.WithAttributes(deliveryStatusAttrs...),
					)

					attempts := status.Attempts
					state := notification.EventDeliveryStatusStateSending
					nextAttempt := (*time.Time)(nil)

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
					errs = append(errs,
						fmt.Errorf("unhandled reconciling state [namespace=%s notification.delivery_status.id=%s notification.delivery_status.state=%s]",
							status.Namespace, status.ID, status.State.String()),
					)
				}
			case notification.EventDeliveryStatusStateResending:
				// Note: keep error local, so we do not break the reconcile logic
				// for delivery status in 'PENDING' or 'SENDING' states.
				err := h.resendWebhookMessage(ctx, event, &status)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to resend webhook message: %w", err))
				}

				// Ignore any error from the resend operation.
				input = &notification.UpdateEventDeliveryStatusInput{
					NamespacedID: status.NamespacedID,
					State:        notification.EventDeliveryStatusStateSending,
					Annotations:  status.Annotations,
					NextAttempt:  lo.ToPtr(now),
					Attempts:     status.Attempts,
				}
			case notification.EventDeliveryStatusStateSending:
				// Check if the delivery status is sending for too long.
				if now.Sub(status.CreatedAt) > h.sendingTimeout {
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

					if now.Sub(stuckInSendingSince) > time.Hour {
						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStatePending,
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}
					}
				case msg != nil:
					// If the event has just been dispatched to the provider, the delivery status(es) won't be available yet.
					// Check if the message timestamp is after the time the delivery status was last updated.
					// If so, skip reconciling the delivery status, the next reconciliation attempt will hopefully fetch the delivery status(es),
					// which will update the delivery status accordingly.
					if msg.DeliveryStatuses == nil && msg.Timestamp.After(status.UpdatedAt) {
						// Skip reconciling the delivery status since the message has been just dispatched to the provider.
						break
					}

					span.AddEvent("fetched webhook message from provider",
						trace.WithAttributes(spanAttrs...),
						trace.WithAttributes(deliveryStatusAttrs...),
					)

					wh, ok := webhooksByChannelID[status.ChannelID]
					if !ok {
						h.logger.ErrorContext(ctx, "notification channel for delivery status does not exist at webhook provider. it means its state is out of sync",
							"namespace", event.Namespace,
							"notification.event.id", event.ID,
							"notification.delivery_status.id", status.ID,
							"notification.channel.id", status.ChannelID,
						)

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       ErrSystemChannelNotFound.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}

						break
					}

					if !lo.Contains(wh.Channels, event.Rule.ID) {
						h.logger.ErrorContext(ctx, "notification rule is not associated with notification channel for delivery status at webhook provider. it means its state is out of sync",
							"namespace", event.Namespace,
							"notification.event.id", event.ID,
							"notification.delivery_status.id", status.ID,
							"notification.channel.id", status.ChannelID,
							"notification.rule.id", event.Rule.ID,
						)

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       ErrSystemRuleNotAssignToChannel.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}

						break
					}

					if wh.Disabled {
						h.logger.WarnContext(ctx, "notification channel for delivery status is disabled at webhook provider. it means its state is out of sync",
							"namespace", event.Namespace,
							"notification.event.id", event.ID,
							"notification.delivery_status.id", status.ID,
							"notification.channel.id", status.ChannelID,
						)

						input = &notification.UpdateEventDeliveryStatusInput{
							NamespacedID: status.NamespacedID,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       ErrSystemChannelDisabled.Error(),
							Annotations:  status.Annotations,
							NextAttempt:  nil,
							Attempts:     status.Attempts,
						}

						break
					}

					// Get the message delivery status for the current channel.
					msgStatusByChannel := getDeliveryStatusByChannelID(lo.FromPtr(msg.DeliveryStatuses), status.ChannelID)

					// If message delivery status not found, skip reconciling the delivery status as
					// the provider might not populate the delivery status yet. The delivery status will be eventually
					// reconciled either by the information provided by the provider or by setting the delivery status
					// to 'FAILED' after the sending timeout is reached.
					if msgStatusByChannel == nil {
						break
					}

					// Check if attempts are synchronized between the webhook provider and OpenMeter.
					// This check is required to avoid transitioning the delivery status to its final state
					// ('SUCCESS' or 'FAILED') before all the attempts are collected from the provider.
					// It is possible that the provider reports that the message delivery status is 'SUCCESS' or 'FAILED'
					// while the corresponding delivery attempt is not yet available. Therefore, we need to keep reconciling
					// the delivery status until all the attempts are collected.
					if msgStatusByChannel.State == notification.EventDeliveryStatusStateSuccess ||
						msgStatusByChannel.State == notification.EventDeliveryStatusStateFailed {
						if len(status.Attempts) >= len(msgStatusByChannel.Attempts) {
							// The list of delivery attempts from the message (returned by the provider) must have at least one extra item
							// compared to the list of delivery status has when the message delivery status is in one of the final states.
							// Wait until the next reconciliation attempt to reconcile the delivery status to ensure we collected all the
							// delivery attempts from the provider.
							break
						}
					}

					input = &notification.UpdateEventDeliveryStatusInput{
						NamespacedID: status.NamespacedID,
						State:        msgStatusByChannel.State,
						Annotations:  status.Annotations.Merge(msg.Annotations),
						NextAttempt:  msgStatusByChannel.NextAttempt,
						Attempts:     msgStatusByChannel.Attempts,
					}
				default:
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

			span.AddEvent("updating delivery status", trace.WithAttributes(spanAttrs...))
			h.logger.DebugContext(ctx, "updating delivery status", "update", input)

			_, err = h.repo.UpdateEventDeliveryStatus(ctx, *input)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to update event delivery [namespace=%s notification.delivery_status.id=%s]: %w", status.Namespace, status.ID, err))
			}
		}

		return errors.Join(errs...)
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

func (h *Handler) resendWebhookMessage(ctx context.Context, event *notification.Event, status *notification.EventDeliveryStatus) error {
	fn := func(ctx context.Context) error {
		if event == nil {
			return fmt.Errorf("event must not be nil")
		}

		if status == nil {
			return fmt.Errorf("delivery staus must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.namespace", event.Namespace),
			attribute.Stringer("notification.event.type", event.Type),
			attribute.String("notification.event.id", event.ID),
			attribute.String("notification.channel.id", status.ChannelID),
		}

		span.SetAttributes(spanAttrs...)

		span.AddEvent("resending webhook message on channel", trace.WithAttributes(spanAttrs...))

		err := h.webhook.ResendMessage(ctx, webhook.ResendMessageInput{
			Namespace: event.Namespace,
			EventID:   event.ID,
			ChannelID: status.ChannelID,
		})
		if err != nil {
			return fmt.Errorf("failed to resend webhook message [namespace=%s notification.event.id=%s notification.channel.id=%s]: %w",
				event.Namespace, event.ID, status.ChannelID, err)
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.resend_webhook_message").Wrap(fn)
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
				DeliveryStatusByChannelID: event.Rule.ID,
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

		payload, err := eventAsPayload(event)
		if err != nil {
			return nil, webhook.NewValidationError(
				fmt.Errorf("failed to serialize webhook message payload: %w", err),
			)
		}

		msg, err := h.webhook.SendMessage(ctx, webhook.SendMessageInput{
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
