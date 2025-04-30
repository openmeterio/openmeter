package subscriptionaddon

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "subscription"
)

type subscriptionAddonEvent struct {
	SubscriptionAddon `json:",inline"`
	CustomerID        string  `json:"customerId"`
	UserID            *string `json:"userId,omitempty"`
}

func (s subscriptionAddonEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(s.SubscriptionAddon.Namespace, metadata.EntitySubscriptionAddon, s.SubscriptionAddon.ID),
		Subject: metadata.ComposeResourcePath(s.SubscriptionAddon.Namespace, metadata.EntityCustomer, s.CustomerID),
	}
}

func (s subscriptionAddonEvent) Validate() error {
	return nil
}

func NewCreatedEvent(ctx context.Context, addon *SubscriptionAddon, customerID string) CreatedEvent {
	return CreatedEvent{
		SubscriptionAddon: *addon,
		CustomerID:        customerID,
		UserID:            session.GetSessionUserID(ctx),
	}
}

type CreatedEvent subscriptionAddonEvent

var (
	_ marshaler.Event = CreatedEvent{}

	subscriptionAddonCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscriptionaddon.created",
		Version:   "v1",
	})
)

func (s CreatedEvent) EventName() string {
	return subscriptionAddonCreatedEventName
}

func (s CreatedEvent) EventMetadata() metadata.EventMetadata {
	return subscriptionAddonEvent(s).EventMetadata()
}

func (s CreatedEvent) Validate() error {
	return subscriptionAddonEvent(s).Validate()
}

func NewUpdatedEvent(ctx context.Context, addon *SubscriptionAddon, customerID string) UpdatedEvent {
	return UpdatedEvent{
		SubscriptionAddon: *addon,
		CustomerID:        customerID,
		UserID:            session.GetSessionUserID(ctx),
	}
}

type UpdatedEvent subscriptionAddonEvent

var (
	_ marshaler.Event = UpdatedEvent{}

	subscriptionAddonUpdatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscriptionaddon.updated",
		Version:   "v1",
	})
)

func (s UpdatedEvent) EventName() string {
	return subscriptionAddonUpdatedEventName
}

func (s UpdatedEvent) EventMetadata() metadata.EventMetadata {
	return subscriptionAddonEvent(s).EventMetadata()
}

func (s UpdatedEvent) Validate() error {
	return subscriptionAddonEvent(s).Validate()
}
