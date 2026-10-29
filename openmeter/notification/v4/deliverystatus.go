package v4

import (
	"encoding/json"
	"time"
)

type EventDeliveryStatusState string

const (
	EventDeliveryStatusStatePending   EventDeliveryStatusState = "pending"
	EventDeliveryStatusStateSending   EventDeliveryStatusState = "sending"
	EventDeliveryStatusStateSuccess   EventDeliveryStatusState = "success"
	EventDeliveryStatusStateFailed    EventDeliveryStatusState = "failed"
	EventDeliveryStatusStateResending EventDeliveryStatusState = "resending"
)

// EventDeliveryAttempt is a Provider observation stored with the Event-Channel
// delivery status. Response is optional Provider-owned JSON; Notification stores
// it without interpreting it. Any truncation marker belongs inside Response.
type EventDeliveryAttempt struct {
	State        EventDeliveryStatusState
	Timestamp    time.Time
	Response     json.RawMessage
	ProviderData map[string]string
}
