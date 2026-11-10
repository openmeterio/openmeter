package adapter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ChannelFromDBEntity(e db.NotificationChannel) (*notification.Channel, error) {
	config, err := fromDBChannelConfig(e.Type, e.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid notification channel config [namespace=%s channel.id=%s]: %w", e.Namespace, e.ID, err)
	}

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
		Config:   config,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}, nil
}

func RuleFromDBEntity(e db.NotificationRule) (*notification.Rule, error) {
	config, err := fromDBRuleConfig(e.Type, e.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid notification rule config [namespace=%s rule.id=%s]: %w", e.Namespace, e.ID, err)
	}

	var channels []notification.Channel
	if len(e.Edges.Channels) > 0 {
		for _, channel := range e.Edges.Channels {
			if channel == nil {
				continue
			}

			mapped, err := ChannelFromDBEntity(*channel)
			if err != nil {
				return nil, fmt.Errorf("failed to map notification rule channel [rule.id=%s]: %w", e.ID, err)
			}

			channels = append(channels, *mapped)
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
		Config:   config,
		Channels: channels,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}, nil
}

func EventFromDBEntity(e db.NotificationEvent) (*notification.Event, error) {
	payload, err := eventPayloadFromJSON(e.Payload)
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

			mapped, err := EventDeliveryStatusFromDBEntity(*status)
			if err != nil {
				return nil, fmt.Errorf("failed to map notification event delivery status [event.id=%s]: %w", e.ID, err)
			}

			statuses = append(statuses, *mapped)
		}
	}

	ruleRow, err := e.Edges.RulesOrErr()
	if err != nil {
		return nil, err
	}

	rule, err := RuleFromDBEntity(*ruleRow)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification event rule [event.id=%s]: %w", e.ID, err)
	}

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
	var meta notification.EventPayloadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload meta: %w", err)
	}

	switch meta.Type {
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		if meta.Version != notification.EventPayloadVersionCurrent {
			return notification.EventPayload{}, fmt.Errorf("unsupported notification event payload version: %d", meta.Version)
		}
	}

	var payload notification.EventPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload: %w", err)
	}

	return payload, nil
}

func EventDeliveryStatusFromDBEntity(e db.NotificationEventDeliveryStatus) (*notification.EventDeliveryStatus, error) {
	var attempts []notification.EventDeliveryAttempt
	if len(e.Attempts) > 0 {
		if err := json.Unmarshal(e.Attempts, &attempts); err != nil {
			return nil, fmt.Errorf("invalid notification event delivery attempts [namespace=%s delivery_status.id=%s]: %w", e.Namespace, e.ID, err)
		}
	}

	state, _ := deliveryStateSpellings(e.State)

	return &notification.EventDeliveryStatus{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ChannelID: e.ChannelID,
		EventID:   e.EventID,
		State:     state,
		Reason:    e.Reason,
		CreatedAt: e.CreatedAt.UTC(),
		UpdatedAt: e.UpdatedAt.UTC(),

		NextAttempt: func() *time.Time {
			if e.NextAttemptAt == nil {
				return nil
			}

			return lo.ToPtr(lo.FromPtr(e.NextAttemptAt).UTC())
		}(),
		Attempts: attempts,

		Annotations: e.Annotations,
	}, nil
}
