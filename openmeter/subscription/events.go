package subscription

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "subscription"
)

type viewEvent struct {
	SubscriptionView `json:",inline"`
	UserID           *string `json:"userId,omitempty"`
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

// NewCreatedEvent creates a new created event
func NewCreatedEvent(ctx context.Context, view SubscriptionView) CreatedEvent {
	return CreatedEvent{
		SubscriptionView: view,
		UserID:           session.GetSessionUserID(ctx),
	}
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

// NewDeletedEvent creates a new deleted event
func NewDeletedEvent(ctx context.Context, view SubscriptionView) DeletedEvent {
	return DeletedEvent{
		SubscriptionView: view,
		UserID:           session.GetSessionUserID(ctx),
	}
}

type DeletedEvent viewEvent

var (
	_ marshaler.Event = CreatedEvent{}

	subscriptionDeletedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.deleted",
		Version:   "v1",
	})
)

func (s DeletedEvent) EventName() string {
	return subscriptionDeletedEventName
}

func (s DeletedEvent) EventMetadata() metadata.EventMetadata {
	return viewEvent(s).EventMetadata()
}

func (s DeletedEvent) Validate() error {
	return viewEvent(s).Validate()
}

// NewCancelledEvent creates a new deleted event
func NewCancelledEvent(ctx context.Context, view SubscriptionView) CancelledEvent {
	return CancelledEvent{
		SubscriptionView: view,
		UserID:           session.GetSessionUserID(ctx),
	}
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

// NewContinuedEvent creates a new continued event
func NewContinuedEvent(ctx context.Context, view SubscriptionView) ContinuedEvent {
	return ContinuedEvent{
		SubscriptionView: view,
		UserID:           session.GetSessionUserID(ctx),
	}
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

// NewUpdatedEvent creates a new updated event
func NewUpdatedEvent(ctx context.Context, view SubscriptionView) UpdatedEvent {
	return UpdatedEvent{
		UpdatedView: view,
		UserID:      session.GetSessionUserID(ctx),
	}
}

type UpdatedEvent struct {
	// We can consider adding the old version or diff here if needed
	UpdatedView SubscriptionView `json:"updatedView"`
	UserID      *string          `json:"userId,omitempty"`
}

var (
	_ marshaler.Event = UpdatedEvent{}

	subscriptionUpdatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.updated",
		Version:   "v1",
	})
)

func (s UpdatedEvent) EventName() string {
	return subscriptionUpdatedEventName
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

type SubscriptionSyncEvent struct {
	Subscription Subscription `json:"subscription"`
}

func NewSubscriptionSyncEvent(ctx context.Context, sub Subscription) SubscriptionSyncEvent {
	return SubscriptionSyncEvent{
		Subscription: sub,
	}
}

var (
	_ marshaler.Event = SubscriptionSyncEvent{}

	subscriptionSyncEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "subscription.sync",
		Version:   "v1",
	})
)

func (s SubscriptionSyncEvent) EventName() string {
	return subscriptionSyncEventName
}

func (s SubscriptionSyncEvent) Validate() error {
	var errs []error

	if err := s.Subscription.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if s.Subscription.CustomerId == "" {
		errs = append(errs, errors.New("customer id is required"))
	}

	return errors.Join(errs...)
}

func (s SubscriptionSyncEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(s.Subscription.Namespace, metadata.EntitySubscription, s.Subscription.ID),
		Subject: metadata.ComposeResourcePath(s.Subscription.Namespace, metadata.EntityCustomer, s.Subscription.CustomerId),
	}
}
