package billing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The event payload keeps the legacy `taxOverrides` shape for consumers of the system events
// topic; the domain line only carries the Tax Code ID, so the constructors expand it once from
// the loaded Tax Codes. Decoding keeps the payload as it came, so an event never needs a Tax
// Code lookup to be encoded again.

// lineTaxOverrides is the legacy taxOverrides payload of the lines of an event, by line ID.
type lineTaxOverrides map[string]*StoredLineTaxConfig

type newStandardLineTaxOverridesInput struct {
	Lines    []*StandardLine
	TaxCodes productcatalog.TaxCodesByID

	// InvoicingAppType is the type of the invoice's invoicing app. On Stripe invoices the
	// payload's stripe.code is the code the line was billed with, not the live mapping.
	InvoicingAppType app.AppType
}

// Validate requires every Tax Code a line references to be loaded: the payload embeds the Tax
// Code, and an event must not be published with a reference it cannot expand.
func (i newStandardLineTaxOverridesInput) Validate() error {
	var errs []error

	for idx, line := range i.Lines {
		if line == nil {
			errs = append(errs, fmt.Errorf("line [%d] is nil", idx))
			continue
		}

		if line.TaxConfig == nil || line.TaxConfig.TaxCodeID == "" {
			continue
		}

		if _, ok := i.TaxCodes.Get(line.TaxConfig.TaxCodeID); !ok {
			errs = append(errs, fmt.Errorf("line %s: tax code %q not found", line.ID, line.TaxConfig.TaxCodeID))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func newStandardLineTaxOverrides(in newStandardLineTaxOverridesInput) (lineTaxOverrides, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	out := make(lineTaxOverrides, len(in.Lines))

	for _, line := range in.Lines {
		stored, err := NewStoredStandardLineTaxConfig(line.TaxConfig, in.TaxCodes)
		if err != nil {
			return nil, fmt.Errorf("line %s: %w", line.ID, err)
		}

		if stored != nil {
			// Decoding copies stripe.code back into the line's legacy fallback, so it must
			// be what Stripe received, or nothing if the line was billed without a code.
			if in.InvoicingAppType == app.AppTypeStripe {
				stored.Stripe = nil

				if code := line.BilledTaxCode(in.InvoicingAppType); code != "" {
					stored.Stripe = &productcatalog.StripeTaxConfig{Code: code}
				}
			}

			out[line.ID] = stored
		}
	}

	return out, nil
}

func newGatheringLineTaxOverrides(lines []GatheringLine, taxCodes productcatalog.TaxCodesByID) (lineTaxOverrides, error) {
	out := make(lineTaxOverrides, len(lines))

	for _, line := range lines {
		stored, err := NewStoredGatheringLineTaxConfig(line.TaxConfig, taxCodes)
		if err != nil {
			return nil, fmt.Errorf("line %s: %w", line.ID, err)
		}

		if stored != nil {
			out[line.ID] = stored
		}
	}

	return out, nil
}

// forLine returns the payload stored for the line. A line without one (an event that was not
// built by a constructor or decoded) gets only its Tax Code ID and behavior.
func (o lineTaxOverrides) forLine(id string, cfg *productcatalog.TaxCodeConfig) *StoredLineTaxConfig {
	if stored, ok := o[id]; ok {
		return stored
	}

	return newStoredLineTaxConfigWithoutTaxCode(cfg)
}

// eventLineTaxOverridesJSON reads just the legacy payload of a serialized line. The line itself
// is decoded separately and only knows the Tax Code ID and behavior from the same key.
type eventLineTaxOverridesJSON struct {
	ID        string               `json:"id"`
	TaxConfig *StoredLineTaxConfig `json:"taxOverrides,omitempty"`
}

func newLineTaxOverridesFromJSON(lines []eventLineTaxOverridesJSON) lineTaxOverrides {
	out := make(lineTaxOverrides, len(lines))

	for _, line := range lines {
		if line.TaxConfig != nil {
			out[line.ID] = line.TaxConfig
		}
	}

	return out
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
			lines = append(lines, eventStandardLineJSON{
				StandardLine: line,
				TaxConfig:    e.taxOverrides.forLine(line.ID, line.TaxConfig),
			})
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

// UnmarshalJSON keeps each line's legacy taxOverrides payload for encoding. It also fills
// LegacyStripeTaxCode from its stripe.code, which is how events published before lines carried
// the invoicing app tax code still show the code the invoice was billed with.
func (e *EventStandardInvoice) UnmarshalJSON(data []byte) error {
	// plain has no methods, so decoding it does not recurse into this function.
	type plain EventStandardInvoice

	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var wire struct {
		Invoice struct {
			Lines []eventLineTaxOverridesJSON `json:"lines"`
		} `json:"invoice"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	lines := decoded.Invoice.Lines.OrEmpty()
	if len(lines) != len(wire.Invoice.Lines) {
		return fmt.Errorf("invoice has %d lines but %d line tax overrides", len(lines), len(wire.Invoice.Lines))
	}

	for idx, line := range lines {
		if overrides := wire.Invoice.Lines[idx].TaxConfig; overrides != nil && overrides.Stripe != nil {
			line.LegacyStripeTaxCode = overrides.Stripe.Code
		}
	}

	decoded.taxOverrides = newLineTaxOverridesFromJSON(wire.Invoice.Lines)

	*e = EventStandardInvoice(decoded)

	return nil
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
			lines = append(lines, eventGatheringLineJSON{
				GatheringLine: line,
				TaxConfig:     e.taxOverrides.forLine(line.ID, line.TaxConfig),
			})
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

// UnmarshalJSON keeps each line's legacy taxOverrides payload for encoding.
func (e *GatheringInvoiceCreatedEvent) UnmarshalJSON(data []byte) error {
	type plain GatheringInvoiceCreatedEvent

	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var wire struct {
		Invoice struct {
			Lines []eventLineTaxOverridesJSON `json:"lines"`
		} `json:"gatheringInvoice"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	decoded.taxOverrides = newLineTaxOverridesFromJSON(wire.Invoice.Lines)

	*e = GatheringInvoiceCreatedEvent(decoded)

	return nil
}
