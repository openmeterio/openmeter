package repository

import (
	"time"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ChannelFromDBEntity(e db.NotificationChannel) *notification.Channel {
	return &notification.Channel{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
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
		ID:       e.ID,
		Type:     e.Type,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   e.Config,
	}
}

func RuleFromDBEntity(e db.NotificationRule) *notification.Rule {
	return &notification.Rule{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
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
		ID:       e.ID,
		Type:     e.Type,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   e.Config,
	}
}

func EventFromDBEntity(e db.NotificationEvent) *notification.Event {
	return &notification.Event{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ID:             e.ID,
		Type:           e.Type,
		CreatedAt:      e.CreatedAt,
		DeliveryStatus: nil,
		// FIXME:
		Payload: notification.EventPayload{},
		// FIXME:
		Rule: notification.Rule{},
	}
}

func EventDeliveryStatusFromDBEntity(e db.NotificationEventDeliveryStatus) *notification.EventDeliveryStatus {
	return &notification.EventDeliveryStatus{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ID:        e.ID,
		ChannelID: e.ChannelID,
		State:     e.State,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
