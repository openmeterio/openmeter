package billing

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// The event payload keeps the legacy `taxOverrides` shape for consumers of the system events
// topic; the domain line only carries the Tax Code ID, so we re-expand it from the loaded codes.

// validateTaxCodesPresent ensures every referenced Tax Code ID is loaded, so marshaling cannot fail later.
func validateTaxCodesPresent(configs []*productcatalog.TaxCodeConfig, taxCodes productcatalog.TaxCodesByID) error {
	for _, cfg := range configs {
		if cfg == nil || cfg.TaxCodeID == "" {
			continue
		}

		if _, ok := taxCodes.Get(cfg.TaxCodeID); !ok {
			return fmt.Errorf("tax code %q not found", cfg.TaxCodeID)
		}
	}

	return nil
}

type eventStandardLineJSON struct {
	*StandardLine
	TaxConfig *StoredLineTaxConfig `json:"taxOverrides,omitempty"`
}

type eventStandardInvoiceJSON struct {
	StandardInvoice
	Lines json.RawMessage `json:"lines"`
}

func (e EventStandardInvoice) MarshalJSON() ([]byte, error) {
	out := eventStandardInvoiceJSON{StandardInvoice: e.Invoice, Lines: json.RawMessage("null")}

	if e.Invoice.Lines.IsPresent() {
		lines := make([]eventStandardLineJSON, 0, len(e.Invoice.Lines.OrEmpty()))

		for _, line := range e.Invoice.Lines.OrEmpty() {
			taxConfig, err := NewStoredStandardLineTaxConfig(line.TaxConfig, e.taxCodes)
			if err != nil {
				return nil, fmt.Errorf("line %s: %w", line.ID, err)
			}

			lines = append(lines, eventStandardLineJSON{StandardLine: line, TaxConfig: taxConfig})
		}

		raw, err := json.Marshal(lines)
		if err != nil {
			return nil, err
		}

		out.Lines = raw
	}

	return json.Marshal(struct {
		Invoice eventStandardInvoiceJSON `json:"invoice"`
		Apps    InvoiceApps              `json:"apps,omitempty"`
	}{Invoice: out, Apps: e.Apps})
}

type eventGatheringLineJSON struct {
	GatheringLine
	TaxConfig *StoredLineTaxConfig `json:"taxOverrides,omitempty"`
}

type eventGatheringInvoiceJSON struct {
	GatheringInvoice
	Lines json.RawMessage `json:"lines,omitempty"`
}

func (e GatheringInvoiceCreatedEvent) MarshalJSON() ([]byte, error) {
	out := eventGatheringInvoiceJSON{GatheringInvoice: e.Invoice, Lines: json.RawMessage("null")}

	if e.Invoice.Lines.IsPresent() {
		lines := make([]eventGatheringLineJSON, 0, len(e.Invoice.Lines.OrEmpty()))

		for _, line := range e.Invoice.Lines.OrEmpty() {
			taxConfig, err := NewStoredGatheringLineTaxConfig(line.TaxConfig, e.taxCodes)
			if err != nil {
				return nil, fmt.Errorf("line %s: %w", line.ID, err)
			}

			lines = append(lines, eventGatheringLineJSON{GatheringLine: line, TaxConfig: taxConfig})
		}

		raw, err := json.Marshal(lines)
		if err != nil {
			return nil, err
		}

		out.Lines = raw
	}

	return json.Marshal(struct {
		Invoice eventGatheringInvoiceJSON `json:"gatheringInvoice"`
	}{Invoice: out})
}
