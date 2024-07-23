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
