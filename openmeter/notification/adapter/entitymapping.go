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
	payload := notification.EventPayload{}
	if err := json.Unmarshal([]byte(e.Payload), &payload); err != nil {
		return nil, fmt.Errorf("failed to serialize notification event payload: %w", err)
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
