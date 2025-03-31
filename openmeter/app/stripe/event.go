package appstripe

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	AppEventSubsystem           metadata.EventSubsystem = "app.stripe"
	AppCheckoutSessionEventName metadata.EventName      = "app.stripe.checkout_session.created"
)

// NewAppCheckoutSessionEvent creates a new checkout session event
func NewAppCheckoutSessionEvent(ctx context.Context, namespace string, sessionID string, appID string, customerID string) AppCheckoutSessionEvent {
	return AppCheckoutSessionEvent{
		Namespace:  namespace,
		SessionID:  sessionID,
		AppID:      appID,
		CustomerID: customerID,
		UserID:     session.GetSessionUserID(ctx),
	}
}

// AppCheckoutSessionEvent is an event that is emitted when a checkout session is created
type AppCheckoutSessionEvent struct {
	SessionID  string  `json:"-"`
	Namespace  string  `json:"namespace"`
	AppID      string  `json:"appId"`
	CustomerID string  `json:"customerId"`
	UserID     *string `json:"userId,omitempty"`
}

func (e AppCheckoutSessionEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: AppEventSubsystem,
		Name:      AppCheckoutSessionEventName,
		Version:   "v1",
	})
}

func (e AppCheckoutSessionEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Namespace, metadata.EntityApp, "stripe", "checkoutSession", e.SessionID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    time.Now(),
	}
}

func (e AppCheckoutSessionEvent) Validate() error {
	if e.AppID == "" {
		return fmt.Errorf("app id is required")
	}

	if e.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	if e.SessionID == "" {
		return fmt.Errorf("session id is required")
	}

	return nil
}
