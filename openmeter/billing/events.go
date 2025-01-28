package billing

import (
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
