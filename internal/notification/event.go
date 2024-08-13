package notification

import (
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Event struct {
	models.NamespacedModel

	// ID is the unique identifier for Event.
	ID string `json:"id"`
	// Type of the notification Event (e.g. entitlements.balance.threshold)
	Type EventType `json:"type"`
	// CreatedAt Timestamp when the notification event was created.
	CreatedAt time.Time `json:"createdAt"`
	// DeliveryStatus defines the delivery status of the notification Event per Channel.
	DeliveryStatus []EventDeliveryStatus `json:"deliveryStatus"`
	// Payload is the actual payload sent to Channel as part of the notification Event.
	Payload EventPayload `json:"payload"`
	// Rule defines the notification Rule that generated this Event.
	Rule Rule `json:"rule"`
}

const (
	EventTypeBalanceThreshold = EventType(api.EntitlementsBalanceThreshold)
)

type EventType api.NotificationEventType

func (t EventType) Values() []string {
	return []string{
		string(EventTypeBalanceThreshold),
	}
}

type EventPayloadMeta struct {
	Type EventType `json:"type"`
}

// EventPayload is a union type capturing payload for all EventType of Events.
type EventPayload struct {
	EventPayloadMeta

	// Balance Threshold
	BalanceThreshold BalanceThresholdPayload `json:"balanceThreshold"`
}

type BalanceThresholdPayload struct {
	Entitlement api.EntitlementMetered                    `json:"entitlement"`
	Feature     api.Feature                               `json:"feature"`
	Subject     api.Subject                               `json:"subject"`
	Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
	Value       api.EntitlementValue                      `json:"value"`
}

const (
	EventDeliveryStatusStateSuccess = EventDeliveryStatusState(api.SUCCESS)
	EventDeliveryStatusStateFailed  = EventDeliveryStatusState(api.FAILED)
	EventDeliveryStatusStateSending = EventDeliveryStatusState(api.SENDING)
	EventDeliveryStatusStatePending = EventDeliveryStatusState(api.PENDING)
)

type EventDeliveryStatusState string

func (e EventDeliveryStatusState) Values() []string {
	return []string{
		string(EventDeliveryStatusStateSuccess),
		string(EventDeliveryStatusStateFailed),
		string(EventDeliveryStatusStateSending),
		string(EventDeliveryStatusStatePending),
	}
}

type EventDeliveryStatus struct {
	models.NamespacedModel

	// ID is the unique identifier for Event.
	ID string `json:"id"`
	// EventID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID string `json:"eventId"`

	ChannelID string                   `json:"channelId"`
	State     EventDeliveryStatusState `json:"state"`
	Reason    string                   `json:"reason,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt,omitempty"`
}
