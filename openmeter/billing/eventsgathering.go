package billing

import "github.com/openmeterio/openmeter/openmeter/event/metadata"

type GatheringInvoiceCreatedEvent struct {
	Invoice GatheringInvoice `json:"gatheringInvoice"`
}

func (e GatheringInvoiceCreatedEvent) Validate() error {
	return e.Invoice.Validate()
}

func NewGatheringInvoiceCreatedEvent(invoice GatheringInvoice) GatheringInvoiceCreatedEvent {
	return GatheringInvoiceCreatedEvent{Invoice: invoice}
}

func (e GatheringInvoiceCreatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "gathering.invoice.created",
		Version:   "v1",
	})
}

func (e GatheringInvoiceCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityGatheringInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.Invoice.CustomerID),
	}
}
