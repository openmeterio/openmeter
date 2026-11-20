package billing

import (
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// GatheringInvoiceCreatedEvent must be built with its constructor, which loads the lines' Tax
// Codes. A payload decoded from JSON cannot be marshaled again, because the Tax Codes are not
// part of it.
type GatheringInvoiceCreatedEvent struct {
	Invoice GatheringInvoice `json:"gatheringInvoice"`

	// taxCodes are used only to render the legacy taxOverrides payload.
	taxCodes map[string]*taxcode.TaxCode
}

func (e GatheringInvoiceCreatedEvent) Validate() error {
	return e.Invoice.Validate()
}

func NewGatheringInvoiceCreatedEvent(invoice GatheringInvoice, taxCodes map[string]*taxcode.TaxCode) (GatheringInvoiceCreatedEvent, error) {
	configs := make([]*productcatalog.TaxCodeConfig, 0, len(invoice.Lines.OrEmpty()))
	for _, line := range invoice.Lines.OrEmpty() {
		configs = append(configs, line.TaxConfig)
	}

	if err := validateTaxCodesPresent(configs, taxCodes); err != nil {
		return GatheringInvoiceCreatedEvent{}, err
	}

	return GatheringInvoiceCreatedEvent{Invoice: invoice, taxCodes: taxCodes}, nil
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
