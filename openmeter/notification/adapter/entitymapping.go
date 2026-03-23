package adapter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttp "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ChannelFromDBEntity(e db.NotificationChannel) *notification.Channel {
	return &notification.Channel{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt.UTC(),
			UpdatedAt: e.UpdatedAt.UTC(),
			DeletedAt: func() *time.Time {
				if e.DeletedAt == nil {
					return nil
				}

				deletedAt := e.DeletedAt.UTC()

				return &deletedAt
			}(),
		},
		Type:     e.Type,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   e.Config,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}
}

func RuleFromDBEntity(e db.NotificationRule) *notification.Rule {
	var channels []notification.Channel
	if len(e.Edges.Channels) > 0 {
		for _, channel := range e.Edges.Channels {
			if channel == nil {
				continue
			}

			channels = append(channels, *ChannelFromDBEntity(*channel))
		}
	}

	return &notification.Rule{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt.UTC(),
			UpdatedAt: e.UpdatedAt.UTC(),
			DeletedAt: func() *time.Time {
				if e.DeletedAt == nil {
					return nil
				}

				deletedAt := e.DeletedAt.UTC()

				return &deletedAt
			}(),
		},
		Type:     e.Type,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   e.Config,
		Channels: channels,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}
}

func EventFromDBEntity(e db.NotificationEvent) (*notification.Event, error) {
	payload, err := eventPayloadFromJSON([]byte(e.Payload))
	if err != nil {
		return nil, err
	}

	var statuses []notification.EventDeliveryStatus
	if len(e.Edges.DeliveryStatuses) > 0 {
		statuses = make([]notification.EventDeliveryStatus, 0, len(e.Edges.DeliveryStatuses))
		for _, status := range e.Edges.DeliveryStatuses {
			if status == nil {
				continue
			}

			statuses = append(statuses, *EventDeliveryStatusFromDBEntity(*status))
		}
	}

	ruleRow, err := e.Edges.RulesOrErr()
	if err != nil {
		return nil, err
	}
	rule := RuleFromDBEntity(*ruleRow)

	return &notification.Event{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		Type:           e.Type,
		CreatedAt:      e.CreatedAt.UTC(),
		Payload:        payload,
		Rule:           *rule,
		DeliveryStatus: statuses,
		Annotations:    e.Annotations,
	}, nil
}

func eventPayloadFromJSON(data []byte) (notification.EventPayload, error) {
	// First pass: read-only meta to get "type + version"
	meta := notification.EventPayloadMeta{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload meta: %w", err)
	}

	payload := notification.EventPayload{}

	// Second pass: version-aware deserialization for invoice types
	switch meta.Type {
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		switch meta.Version {
		case notification.EventPayloadVersionLegacy: // v0 legacy: stored as billing.EventStandardInvoice
			var v0 struct {
				notification.EventPayloadMeta
				Invoice *billing.EventStandardInvoice `json:"invoice,omitempty"`
			}

			if err := json.Unmarshal(data, &v0); err != nil {
				return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload to legacy v0 schema: %w", err)
			}

			if v0.Invoice == nil {
				return notification.EventPayload{}, fmt.Errorf("missing invoice in legacy event payload")
			}

			apiInvoice, err := billinghttp.MapEventInvoiceToAPI(*v0.Invoice)
			if err != nil {
				return notification.EventPayload{}, fmt.Errorf("failed to map legacy event invoice to API: %w", err)
			}

			payload.EventPayloadMeta = notification.EventPayloadMeta{
				Type:    meta.Type,
				Version: notification.EventPayloadVersionCurrent,
			}
			payload.Invoice = &notification.InvoicePayload{Invoice: apiInvoice}

		case notification.EventPayloadVersionCurrent: // v1: stored as api.Invoice directly
			if err := json.Unmarshal(data, &payload); err != nil {
				return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload: %w", err)
			}
		default:
			return notification.EventPayload{}, fmt.Errorf("unsupported notification event payload version: %d", meta.Version)
		}
	default:
		// all non-invoice types: unaffected, single-pass as before
		if err := json.Unmarshal(data, &payload); err != nil {
			return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload: %w", err)
		}
	}

	return payload, nil
}

func EventDeliveryStatusFromDBEntity(e db.NotificationEventDeliveryStatus) *notification.EventDeliveryStatus {
	return &notification.EventDeliveryStatus{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ChannelID: e.ChannelID,
		EventID:   e.EventID,
		State:     e.State,
		Reason:    e.Reason,
		CreatedAt: e.CreatedAt.UTC(),
		UpdatedAt: e.UpdatedAt.UTC(),

		NextAttempt: func() *time.Time {
			if e.NextAttemptAt == nil {
				return nil
			}

			return lo.ToPtr(lo.FromPtr(e.NextAttemptAt).UTC())
		}(),
		Attempts: e.Attempts,

		Annotations: e.Annotations,
	}
}
