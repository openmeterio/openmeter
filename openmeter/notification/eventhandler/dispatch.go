package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
)

func eventAsPayload(event *notification.Event) (map[string]interface{}, error) {
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

	var m map[string]interface{}

	m, err = notification.PayloadToMapInterface(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to cast event payload: %w", err)
	}

	return m, nil
}

func (h *Handler) dispatchWebhook(ctx context.Context, event *notification.Event) error {
	fn := func(ctx context.Context) error {
		if event == nil {
			return fmt.Errorf("event is nil")
		}

		payload, err := eventAsPayload(event)
		if err != nil {
			return err
		}

		sendIn := webhook.SendMessageInput{
			Namespace: event.Namespace,
			EventID:   event.ID,
			EventType: event.Type.String(),
			Channels:  []string{event.Rule.ID},
			Payload:   payload,
		}

		logger := h.logger.With("eventID", event.ID, "eventType", event.Type)

		var stateReason string

		state := notification.EventDeliveryStatusStateSuccess

		_, err = h.webhook.SendMessage(ctx, sendIn)
		if err != nil {
			logger.ErrorContext(ctx, "failed to send webhook message: error returned by webhook service", "error", err)

			stateReason = "failed to send webhook message: error returned by webhook service"

			state = notification.EventDeliveryStatusStateFailed
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.app_id", sendIn.Namespace),
			attribute.String("svix.event_type", sendIn.EventType),
			attribute.String("svix.event_id", sendIn.EventID),
		}

		span.SetAttributes(spanAttrs...)

		for _, channelID := range notification.ChannelIDsByType(event.Rule.Channels, notification.ChannelTypeWebhook) {
			span.AddEvent("updating event delivery status", trace.WithAttributes(spanAttrs...),
				trace.WithAttributes(attribute.String("notification.event.id", event.ID)),
				trace.WithAttributes(attribute.String("notification.channel.id", channelID)),
			)

			_, err = h.repo.UpdateEventDeliveryStatus(ctx, notification.UpdateEventDeliveryStatusInput{
				NamespacedModel: models.NamespacedModel{
					Namespace: event.Namespace,
				},
				State:     state,
				Reason:    stateReason,
				EventID:   event.ID,
				ChannelID: channelID,
			})
			if err != nil {
				return fmt.Errorf("failed to update event delivery: %w", err)
			}
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch_webhook").Wrap(fn)
}

func (h *Handler) dispatch(ctx context.Context, event *notification.Event) error {
	var errs []error

	for _, channelType := range notification.ChannelTypes(event.Rule.Channels) {
		var err error

		switch channelType {
		case notification.ChannelTypeWebhook:
			err = h.dispatchWebhook(ctx, event)
		default:
			err = fmt.Errorf("unknown channel type: %s", channelType)
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (h *Handler) Dispatch(ctx context.Context, event *notification.Event) error {
	spanLink := trace.LinkFromContext(ctx)

	fn := func(ctx context.Context) error {
		return h.dispatch(ctx, event)
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("notification event handler panicked",
					"error", err,
					"code.stacktrace", string(debug.Stack()))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout)
		defer cancel()

		tracerOpts := []trace.SpanStartOption{
			trace.WithNewRoot(),
			trace.WithLinks(spanLink),
		}

		err := tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", tracerOpts...).Wrap(fn)
		if err != nil {
			h.logger.WarnContext(ctx, "failed to dispatch event", "eventID", event.ID, "error", err)
		}
	}()

	return nil
}
