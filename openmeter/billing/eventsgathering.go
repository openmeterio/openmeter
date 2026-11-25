package billing

import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// GatheringInvoiceCreatedEvent carries the legacy per-line taxOverrides payload the constructor
// built from the lines' Tax Codes, so encoding never needs a lookup and a decoded event encodes
// back to the same JSON.
type GatheringInvoiceCreatedEvent struct {
	Invoice GatheringInvoice `json:"gatheringInvoice"`

	// taxOverrides is the legacy taxOverrides payload of each line, by line ID. Lines missing
	// from it encode only their Tax Code ID and behavior.
	taxOverrides lineTaxOverrides
}

func (e GatheringInvoiceCreatedEvent) Validate() error {
	return e.Invoice.Validate()
}

func NewGatheringInvoiceCreatedEvent(invoice GatheringInvoice, taxCodes productcatalog.TaxCodesByID) (GatheringInvoiceCreatedEvent, error) {
	taxOverrides, err := newGatheringLineTaxOverrides(invoice.Lines.OrEmpty(), taxCodes)
	if err != nil {
		return GatheringInvoiceCreatedEvent{}, err
	}

	return GatheringInvoiceCreatedEvent{Invoice: invoice, taxOverrides: taxOverrides}, nil
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
