package subscription

import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "subscription"
)

type viewEvent struct {
	SubscriptionView `json:",inline"`
}

func (s viewEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(s.Subscription.Namespace, metadata.EntitySubscription, s.Subscription.ID),
		Subject: metadata.ComposeResourcePath(s.Subscription.Namespace, metadata.EntityCustomer, s.Customer.ID),
	}
}

func (s viewEvent) Validate() error {
	return s.SubscriptionView.Validate(true)
}

type CreatedEvent viewEvent

var (
	_ marshaler.Event = CreatedEvent{}

	subscriptionCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.created",
		Version:   "v1",
	})
)

func (s CreatedEvent) EventName() string {
	return subscriptionCreatedEventName
}

func (s CreatedEvent) EventMetadata() metadata.EventMetadata {
	return viewEvent(s).EventMetadata()
}

func (s CreatedEvent) Validate() error {
	return viewEvent(s).Validate()
}

type CancelledEvent viewEvent

var (
	_ marshaler.Event = CancelledEvent{}

	subscriptionCancelledEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.cancelled",
		Version:   "v1",
	})
)

func (s CancelledEvent) EventName() string {
	return subscriptionCancelledEventName
}

func (s CancelledEvent) EventMetadata() metadata.EventMetadata {
	return viewEvent(s).EventMetadata()
}

func (s CancelledEvent) Validate() error {
	return viewEvent(s).Validate()
}

type ContinuedEvent viewEvent

var (
	_ marshaler.Event = ContinuedEvent{}

	subscriptionContinuedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.continued",
		Version:   "v1",
	})
)

func (s ContinuedEvent) EventName() string {
	return subscriptionContinuedEventName
}

func (s ContinuedEvent) EventMetadata() metadata.EventMetadata {
	return viewEvent(s).EventMetadata()
}

func (s ContinuedEvent) Validate() error {
	return viewEvent(s).Validate()
}

type UpdatedEvent struct {
	// We can consider adding the old version or diff here if needed
	UpdatedView SubscriptionView `json:"updatedView"`
}

var (
	_ marshaler.Event = UpdatedEvent{}

	subscriptionEditedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.edited",
		Version:   "v1",
	})
)

func (s UpdatedEvent) EventName() string {
	return subscriptionEditedEventName
}

func (s UpdatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(s.UpdatedView.Subscription.Namespace, metadata.EntitySubscription, s.UpdatedView.Subscription.ID),
		Subject: metadata.ComposeResourcePath(s.UpdatedView.Subscription.Namespace, metadata.EntityCustomer, s.UpdatedView.Customer.ID),
	}
}

func (s UpdatedEvent) Validate() error {
	return s.UpdatedView.Validate(true)
}
