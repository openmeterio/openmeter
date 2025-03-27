package customer

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	CustomerEventSubsystem  metadata.EventSubsystem = "customer"
	CustomerCreateEventName metadata.EventName      = "customer.created"
	CustomerUpdateEventName metadata.EventName      = "customer.updated"
	CustomerDeleteEventName metadata.EventName      = "customer.deleted"
)

// NewCustomerCreateEvent creates a new customer create event
func NewCustomerCreateEvent(customer Customer) CustomerCreateEvent {
	return CustomerCreateEvent{
		EventEntityMutationPayload: metadata.EventEntityMutationPayload[Customer]{
			Entity:       metadata.EntityCustomer,
			MutationType: metadata.EntityMutationTypeCreate,
			New:          &customer,
		},
	}
}

// CustomerCreateEvent is an event that is emitted when a customer is created
type CustomerCreateEvent struct {
	metadata.EventEntityMutationPayload[Customer] `json:",inline"`
}

func (e CustomerCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerCreateEventName,
		Version:   "v1",
	})
}

func (e CustomerCreateEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		ID:     e.New.ID,
		Source: metadata.ComposeResourcePath(e.New.Namespace, metadata.EntityCustomer, e.New.ID),
		Time:   e.New.CreatedAt,
	}
}

func (e CustomerCreateEvent) Validate() error {
	var errs []error

	if err := e.EventEntityMutationPayload.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer create event: %w", err))
	}

	return errors.Join(errs...)
}

// NewCustomerUpdateEvent creates a new customer update event
func NewCustomerUpdateEvent(new Customer, previous Customer) CustomerUpdateEvent {
	return CustomerUpdateEvent{
		EventEntityMutationPayload: metadata.EventEntityMutationPayload[Customer]{
			Entity:       metadata.EntityCustomer,
			MutationType: metadata.EntityMutationTypeUpdate,
			New:          &new,
			Previous:     &previous,
		},
	}
}

// CustomerUpdateEvent is an event that is emitted when a customer is updated
type CustomerUpdateEvent struct {
	metadata.EventEntityMutationPayload[Customer] `json:",inline"`
}

func (e CustomerUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerUpdateEventName,
		Version:   "v1",
	})
}

func (e CustomerUpdateEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		ID:     metadata.GetMutationEventID(metadata.EntityMutationTypeUpdate, e.Previous.ID, e.New.UpdatedAt),
		Source: metadata.ComposeResourcePath(e.New.Namespace, metadata.EntityCustomer, e.New.ID),
		Time:   e.New.UpdatedAt,
	}
}

func (e CustomerUpdateEvent) Validate() error {
	var errs []error

	if err := e.EventEntityMutationPayload.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer update event: %w", err))
	}

	return errors.Join(errs...)
}

// NewCustomerDeleteEvent creates a new customer delete event
func NewCustomerDeleteEvent(previous Customer, deletedAt time.Time) CustomerDeleteEvent {
	return CustomerDeleteEvent{
		EventEntityMutationPayload: metadata.EventEntityMutationPayload[Customer]{
			Entity:       metadata.EntityCustomer,
			MutationType: metadata.EntityMutationTypeDelete,
			Previous:     &previous,
		},
		deletedAt: deletedAt,
	}
}

// CustomerDeleteEvent is an event that is emitted when a customer is deleted
type CustomerDeleteEvent struct {
	metadata.EventEntityMutationPayload[Customer] `json:",inline"`
	deletedAt                                     time.Time
}

func (e CustomerDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerDeleteEventName,
		Version:   "v1",
	})
}

func (e CustomerDeleteEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		ID:     metadata.GetMutationEventID(metadata.EntityMutationTypeDelete, e.Previous.ID, e.deletedAt),
		Source: metadata.ComposeResourcePath(e.Previous.Namespace, metadata.EntityCustomer, e.Previous.ID),
		Time:   e.deletedAt,
	}
}

func (e CustomerDeleteEvent) Validate() error {
	var errs []error

	if err := e.EventEntityMutationPayload.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer delete event: %w", err))
	}

	return errors.Join(errs...)
}
