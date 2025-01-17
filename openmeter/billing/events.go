package billing

import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	EventSubsystem metadata.EventSubsystem = "billing"
)

type InvoiceCreatedEvent struct {
	Invoice `json:",inline"`
}

func NewInvoiceCreatedEvent(invoice Invoice) InvoiceCreatedEvent {
	return InvoiceCreatedEvent{invoice.RemoveCircularReferences()}
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
	return e.Invoice.Validate()
}
