package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	EventSubsystem metadata.EventSubsystem = "billing"
)

type InvoiceApps struct {
	Tax       app.EventApp `json:"tax"`
	Payment   app.EventApp `json:"payment"`
	Invoicing app.EventApp `json:"invoicing"`
}

func (a InvoiceApps) Validate() error {
	var errs []error

	if err := a.Tax.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax: %w", err))
	}

	if err := a.Payment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment: %w", err))
	}

	if err := a.Invoicing.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoicing: %w", err))
	}

	return errors.Join(errs...)
}

type EventInvoice struct {
	Invoice Invoice     `json:"invoice"`
	Apps    InvoiceApps `json:"apps,omitempty"`
}

func (e EventInvoice) Validate() error {
	var errs []error

	if err := e.Invoice.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := e.Apps.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func NewEventInvoice(invoice Invoice) (EventInvoice, error) {
	// This causes a stack overflow
	payload := invoice.RemoveCircularReferences()

	// Remove the Apps from the payload, as they are not json unmarshallable
	// but either ways, the apps service should be used in workers to acquire
	// an up-to-date app based on the payload.Workflow.AppReferences
	payload.Workflow.Apps = nil

	apps := InvoiceApps{}
	// TODO[later]: Apps are always present, so we should only use the struct without a pointer
	if invoice.Workflow.Apps != nil {
		var err error
		apps.Invoicing, err = app.NewEventApp(invoice.Workflow.Apps.Invoicing)
		if err != nil {
			return EventInvoice{}, err
		}

		apps.Tax, err = app.NewEventApp(invoice.Workflow.Apps.Tax)
		if err != nil {
			return EventInvoice{}, err
		}

		apps.Payment, err = app.NewEventApp(invoice.Workflow.Apps.Payment)
		if err != nil {
			return EventInvoice{}, err
		}
	}

	return EventInvoice{
		Invoice: payload,
		Apps:    apps,
	}, nil
}

type InvoiceCreatedEvent struct {
	EventInvoice `json:",inline"`
}

func NewInvoiceCreatedEvent(invoice Invoice) (InvoiceCreatedEvent, error) {
	eventInvoice, err := NewEventInvoice(invoice)
	if err != nil {
		return InvoiceCreatedEvent{}, err
	}

	return InvoiceCreatedEvent{EventInvoice: eventInvoice}, nil
}

func (e InvoiceCreatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.created",
		Version:   "v2",
	})
}

func (e InvoiceCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.Invoice.Customer.CustomerID),
	}
}

func (e InvoiceCreatedEvent) Validate() error {
	return e.EventInvoice.Validate()
}

type InvoiceUpdatedEvent struct {
	Old EventInvoice `json:"old"`
	New EventInvoice `json:"new"`
}

func NewInvoiceUpdatedEvent(new Invoice, old EventInvoice) (InvoiceUpdatedEvent, error) {
	newEventInvoice, err := NewEventInvoice(new)
	if err != nil {
		return InvoiceUpdatedEvent{}, err
	}

	return InvoiceUpdatedEvent{
		Old: old,
		New: newEventInvoice,
	}, nil
}

func (e InvoiceUpdatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.updated",
		Version:   "v3",
	})
}

func (e InvoiceUpdatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.New.Invoice.Namespace, metadata.EntityInvoice, e.New.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.New.Invoice.Namespace, metadata.EntityCustomer, e.New.Invoice.Customer.CustomerID),
	}
}

func (e InvoiceUpdatedEvent) Validate() error {
	return e.New.Validate()
}

type AdvanceInvoiceEvent struct {
	Invoice    InvoiceID `json:"invoice"`
	CustomerID string    `json:"customer_id"`
}

func (e AdvanceInvoiceEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.advance",
		Version:   "v1",
	})
}

func (e AdvanceInvoiceEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e AdvanceInvoiceEvent) Validate() error {
	if err := e.Invoice.Validate(); err != nil {
		return err
	}

	if e.CustomerID == "" {
		return fmt.Errorf("customer_id cannot be empty")
	}

	return nil
}
