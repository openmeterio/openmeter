package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type InvoiceEventHandler struct {
	Notification notification.Service
	Logger       *slog.Logger
}

func (h *InvoiceEventHandler) Handle(ctx context.Context, event billing.EventStandardInvoice, eventType notification.EventType) error {
	h.Logger.InfoContext(ctx, "handling invoice event", "event.type", eventType)

	// Skip events for gathering invoices
	if event.Invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	// List active rules available for this event type in namespace
	rules, err := h.Notification.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{event.Invoice.Namespace},
		Types:      []notification.EventType{eventType},
		OrderBy:    notification.OrderByID,
		Order:      sortx.OrderDefault,
	})
	if err != nil {
		return fmt.Errorf("failed to list rules for event type [namespace=%s event.type=%s]: %w",
			event.Invoice.Namespace, eventType, err)
	}

	// TODO: it is planned to allow publishing events without active notification rule in order
	// to store them but not sending them to any channel.
	if len(rules.Items) == 0 {
		h.Logger.WarnContext(ctx, "no rules found for event type: skip creating notification event",
			"namespace", event.Invoice.Namespace,
			"event.type", eventType,
		)

		return nil
	}

	payload := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: eventType,
		},
		Invoice: &event,
	}

	for _, rule := range rules.Items {
		if rule.Disabled {
			h.Logger.WarnContext(ctx, "rule is disabled: skip creating notification event",
				"namespace", event.Invoice.Namespace,
				"rule.type", eventType,
				"rule.id", rule.ID,
				"rule.name", rule.Name,
			)

			continue
		}

		notificationEvent, err := h.Notification.CreateEvent(ctx, notification.CreateEventInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: event.Invoice.Namespace,
			},
			Annotations: map[string]interface{}{
				notification.AnnotationEventInvoiceID:     event.Invoice.ID,
				notification.AnnotationEventInvoiceNumber: event.Invoice.Number,
			},
			Type:    eventType,
			Payload: payload,
			RuleID:  rule.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to create notification event for system event [namespace=%s event.type=%s]: %w",
				event.Invoice.Namespace, eventType, err)
		}

		h.Logger.DebugContext(ctx, "created notification event for system event",
			"namespace", event.Invoice.Namespace,
			"event.type", eventType,
			"notification.event.id", notificationEvent.ID,
		)
	}

	return nil
}
