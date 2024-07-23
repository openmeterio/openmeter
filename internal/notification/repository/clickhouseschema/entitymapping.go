package clickhouseschema

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func EventFromDBEntity(e EventDBEntity) (*notification.Event, error) {
	payload := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventType(e.Type),
		},
	}

	rule := notification.Rule{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
	}

	switch notification.EventType(e.Type) {
	case notification.EventTypeBalanceThreshold:
		// FIXME: this does not work
		if err := json.Unmarshal([]byte(e.Payload), &payload.BalanceThreshold); err != nil {
			return nil, fmt.Errorf("failed to unmarshal notification event payload: %w", err)
		}

		if err := json.Unmarshal([]byte(e.Rule), &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal notification rule: %w", err)
		}
	}

	return &notification.Event{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ID:        e.ID,
		Type:      notification.EventType(e.Type),
		CreatedAt: e.CreatedAt.UTC(),
		Payload:   payload,
		Rule:      rule,
	}, nil
}

func CreateEventInputToDBEntity(params notification.CreateEventInput) (*EventDBEntity, error) {
	var payload string
	switch params.Type {
	case notification.EventTypeBalanceThreshold:
		p, err := json.Marshal(params.Payload.BalanceThreshold)
		if err != nil {
			return nil, notification.ValidationError{
				Err: fmt.Errorf("failed to marshal notification event payload: %w", err),
			}
		}

		payload = string(p)
	}

	var rule string
	switch params.Type {
	case notification.EventTypeBalanceThreshold:
		p, err := json.Marshal(params.Rule.Config)
		if err != nil {
			return nil, notification.ValidationError{
				Err: fmt.Errorf("failed to marshal notification event rule: %w", err),
			}
		}

		payload = string(p)
	}

	id, err := ulid.New(ulid.Timestamp(params.CreatedAt), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ulid for event: %w", err)
	}

	return &EventDBEntity{
		Namespace: params.Namespace,
		CreatedAt: params.CreatedAt.UTC(),
		Type:      string(params.Type),
		ID:        id.String(),
		Payload:   payload,
		Rule:      rule,
	}, nil
}

func DeliveryStatusFromDBEntity(e DeliveryStatusDBEntity) (*notification.EventDeliveryStatus, error) {

	var state notification.EventDeliveryStatusState
	switch notification.EventDeliveryStatusState(e.State) {
	case notification.EventDeliveryStatusStateSending:
		state = notification.EventDeliveryStatusStateSending
	case notification.EventDeliveryStatusStateSuccess:
		state = notification.EventDeliveryStatusStateSuccess
	case notification.EventDeliveryStatusStateFailed:
		state = notification.EventDeliveryStatusStateFailed
	default:
		return nil, fmt.Errorf("unknown delivery status state: %s", e.State)

	}

	return &notification.EventDeliveryStatus{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		EventID:   e.EventID,
		ChannelID: e.ChannelID,
		State:     state,
		UpdatedAt: e.Timestamp.UTC(),
	}, nil
}

func CreateEventDeliveryStatusInputToDBEntity(params notification.CreateEventDeliveryStatusInput) (*DeliveryStatusDBEntity, error) {
	return &DeliveryStatusDBEntity{
		Namespace: params.Namespace,
		Timestamp: params.Timestamp,
		EventID:   params.EventID,
		ChannelID: params.ChannelID,
		State:     string(params.State),
	}, nil
}
