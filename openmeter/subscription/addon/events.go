package subscriptionaddon

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "subscriptionaddon"
)

type event struct {
	Customer          customer.Customer `json:"customer"`
	SubscriptionAddon SubscriptionAddon `json:"subscriptionAddon"`
	UserID            *string           `json:"userId,omitempty"`
}

func (s event) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(s.SubscriptionAddon.Namespace, metadata.EntitySubscriptionAddon, s.SubscriptionAddon.ID),
		Subject: metadata.ComposeResourcePath(s.SubscriptionAddon.Namespace, metadata.EntityCustomer, s.Customer.ID),
	}
}

func (s event) Validate() error {
	return nil
}

// NewCreatedEvent creates a new created event
func NewCreatedEvent(ctx context.Context, customer customer.Customer, subscriptionAddon SubscriptionAddon) CreatedEvent {
	return CreatedEvent{
		Customer:          customer,
		SubscriptionAddon: subscriptionAddon,
		UserID:            session.GetSessionUserID(ctx),
	}
}

type CreatedEvent event

var (
	_ marshaler.Event = CreatedEvent{}

	subscriptionCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscriptionaddon.created",
		Version:   "v1",
	})
)

func (s CreatedEvent) EventName() string {
	return subscriptionCreatedEventName
}

func (s CreatedEvent) EventMetadata() metadata.EventMetadata {
	return event(s).EventMetadata()
}

func (s CreatedEvent) Validate() error {
	return event(s).Validate()
}

// NewChangeQuantityEvent creates a new deleted event
func NewChangeQuantityEvent(ctx context.Context, customer customer.Customer, subscriptionAddon SubscriptionAddon) ChangeQuantityEvent {
	return ChangeQuantityEvent{
		Customer:          customer,
		SubscriptionAddon: subscriptionAddon,
		UserID:            session.GetSessionUserID(ctx),
	}
}

type ChangeQuantityEvent event

var (
	_ marshaler.Event = CreatedEvent{}

	subscriptionChangeQuantityEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscriptionaddon.changequantity",
		Version:   "v1",
	})
)

func (s ChangeQuantityEvent) EventName() string {
	return subscriptionChangeQuantityEventName
}

func (s ChangeQuantityEvent) EventMetadata() metadata.EventMetadata {
	return event(s).EventMetadata()
}

func (s ChangeQuantityEvent) Validate() error {
	return event(s).Validate()
}
