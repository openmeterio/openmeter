package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/session"
)

const (
	CustomerEventSubsystem  metadata.EventSubsystem = "customer"
	CustomerCreateEventName metadata.EventName      = "customer.created"
	CustomerUpdateEventName metadata.EventName      = "customer.updated"
	CustomerDeleteEventName metadata.EventName      = "customer.deleted"
)

// NewCustomerCreateEvent creates a new customer create event
func NewCustomerCreateEvent(ctx context.Context, customer *Customer) CustomerCreateEvent {
	return CustomerCreateEvent{
		Customer: customer,
		UserID:   session.GetSessionUserID(ctx),
	}
}

// CustomerCreateEvent is an event that is emitted when a customer is created
type CustomerCreateEvent struct {
	Customer *Customer `json:"customer"`
	UserID   *string   `json:"userId,omitempty"`
}

func (e CustomerCreateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerCreateEventName,
		Version:   "v1",
	})
}

func (e CustomerCreateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Customer.Namespace, metadata.EntityCustomer, e.Customer.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Customer.CreatedAt,
	}
}

func (e CustomerCreateEvent) Validate() error {
	var errs []error

	if e.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if err := e.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	return errors.Join(errs...)
}

// NewCustomerUpdateEvent creates a new customer update event
func NewCustomerUpdateEvent(ctx context.Context, customer *Customer) CustomerUpdateEvent {
	return CustomerUpdateEvent{
		Customer: customer,
		UserID:   session.GetSessionUserID(ctx),
	}
}

// CustomerUpdateEvent is an event that is emitted when a customer is updated
type CustomerUpdateEvent struct {
	Customer *Customer `json:"customer"`
	UserID   *string   `json:"userId,omitempty"`
}

func (e CustomerUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerUpdateEventName,
		Version:   "v1",
	})
}

func (e CustomerUpdateEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Customer.Namespace, metadata.EntityCustomer, e.Customer.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    e.Customer.UpdatedAt,
	}
}

func (e CustomerUpdateEvent) Validate() error {
	var errs []error

	if e.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if err := e.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	return errors.Join(errs...)
}

// NewCustomerDeleteEvent creates a new customer delete event
func NewCustomerDeleteEvent(ctx context.Context, customer *Customer) CustomerDeleteEvent {
	return CustomerDeleteEvent{
		Customer: customer,
		UserID:   session.GetSessionUserID(ctx),
	}
}

// CustomerDeleteEvent is an event that is emitted when a customer is deleted
type CustomerDeleteEvent struct {
	Customer *Customer `json:"customer"`
	UserID   *string   `json:"userId,omitempty"`
}

func (e CustomerDeleteEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: CustomerEventSubsystem,
		Name:      CustomerDeleteEventName,
		Version:   "v1",
	})
}

func (e CustomerDeleteEvent) EventMetadata() metadata.EventMetadata {
	resourcePath := metadata.ComposeResourcePath(e.Customer.Namespace, metadata.EntityCustomer, e.Customer.ID)

	return metadata.EventMetadata{
		ID:      ulid.Make().String(),
		Source:  resourcePath,
		Subject: resourcePath,
		Time:    *e.Customer.DeletedAt,
	}
}

func (e CustomerDeleteEvent) Validate() error {
	var errs []error

	if e.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if e.Customer.DeletedAt == nil {
		return fmt.Errorf("customer deleted at is required")
	}

	if err := e.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	return errors.Join(errs...)
}
