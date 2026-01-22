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

type EventStandardInvoice struct {
	Invoice StandardInvoice `json:"invoice"`
	Apps    InvoiceApps     `json:"apps,omitempty"`
}

func (e EventStandardInvoice) Validate() error {
	var errs []error

	if err := e.Invoice.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := e.Apps.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func NewEventStandardInvoice(invoice StandardInvoice) (EventStandardInvoice, error) {
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
			return EventStandardInvoice{}, err
		}

		apps.Tax, err = app.NewEventApp(invoice.Workflow.Apps.Tax)
		if err != nil {
			return EventStandardInvoice{}, err
		}

		apps.Payment, err = app.NewEventApp(invoice.Workflow.Apps.Payment)
		if err != nil {
			return EventStandardInvoice{}, err
		}
	}

	return EventStandardInvoice{
		Invoice: payload,
		Apps:    apps,
	}, nil
}

type StandardInvoiceCreatedEvent struct {
	EventStandardInvoice `json:",inline"`
}

func NewStandardInvoiceCreatedEvent(invoice StandardInvoice) (StandardInvoiceCreatedEvent, error) {
	eventInvoice, err := NewEventStandardInvoice(invoice)
	if err != nil {
		return StandardInvoiceCreatedEvent{}, err
	}

	return StandardInvoiceCreatedEvent{EventStandardInvoice: eventInvoice}, nil
}

func (e StandardInvoiceCreatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.created",
		Version:   "v2",
	})
}

func (e StandardInvoiceCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.Invoice.Customer.CustomerID),
	}
}

func (e StandardInvoiceCreatedEvent) Validate() error {
	return e.EventStandardInvoice.Validate()
}

type StandardInvoiceUpdatedEvent struct {
	Old EventStandardInvoice `json:"old"`
	New EventStandardInvoice `json:"new"`
}

func NewStandardInvoiceUpdatedEvent(new StandardInvoice, old EventStandardInvoice) (StandardInvoiceUpdatedEvent, error) {
	newEventInvoice, err := NewEventStandardInvoice(new)
	if err != nil {
		return StandardInvoiceUpdatedEvent{}, err
	}

	return StandardInvoiceUpdatedEvent{
		Old: old,
		New: newEventInvoice,
	}, nil
}

func (e StandardInvoiceUpdatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.updated",
		Version:   "v3",
	})
}

func (e StandardInvoiceUpdatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.New.Invoice.Namespace, metadata.EntityInvoice, e.New.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.New.Invoice.Namespace, metadata.EntityCustomer, e.New.Invoice.Customer.CustomerID),
	}
}

func (e StandardInvoiceUpdatedEvent) Validate() error {
	return e.New.Validate()
}

type AdvanceStandardInvoiceEvent struct {
	Invoice    InvoiceID `json:"invoice"`
	CustomerID string    `json:"customer_id"`
}

func (e AdvanceStandardInvoiceEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.advance",
		Version:   "v1",
	})
}

func (e AdvanceStandardInvoiceEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e AdvanceStandardInvoiceEvent) Validate() error {
	if err := e.Invoice.Validate(); err != nil {
		return err
	}

	if e.CustomerID == "" {
		return fmt.Errorf("customer_id cannot be empty")
	}

	return nil
}
