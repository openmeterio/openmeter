// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
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
		Channels: channels,
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
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ID:             e.ID,
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
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ID:        e.ID,
		ChannelID: e.ChannelID,
		EventID:   e.EventID,
		State:     e.State,
		Reason:    e.Reason,
		CreatedAt: e.CreatedAt.UTC(),
		UpdatedAt: e.UpdatedAt.UTC(),
	}
}
