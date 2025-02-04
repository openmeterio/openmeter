package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	EventSubsystem metadata.EventSubsystem = "billing"
)

type EventInvoice Invoice

func NewEventInvoice(invoice Invoice) EventInvoice {
	// This causes a stack overflow
	payload := invoice.RemoveCircularReferences()

	// Remove the Apps from the payload, as they are not json unmarshallable
	// but either ways, the apps service should be used in workers to acquire
	// an up-to-date app based on the payload.Workflow.AppReferences
	payload.Workflow.Apps = nil

	return EventInvoice(payload)
}

type InvoiceCreatedEvent struct {
	EventInvoice `json:",inline"`
}

func NewInvoiceCreatedEvent(invoice Invoice) InvoiceCreatedEvent {
	return InvoiceCreatedEvent{NewEventInvoice(invoice)}
}

func (e InvoiceCreatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.created",
		Version:   "v1",
	})
}

func (e InvoiceCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace, metadata.EntityInvoice, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.Customer.CustomerID),
	}
}

func (e InvoiceCreatedEvent) Validate() error {
	return e.EventInvoice.Validate()
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
