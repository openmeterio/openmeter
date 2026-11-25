package invoicecalc

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// SnapshotTaxConfigIntoLines merges the invoice's default tax config (deps.DefaultTaxConfig) into
// each line: the default's Tax Code ID and behavior fill in what the line leaves unset.
// Skipped for gathering invoices — snapshotting only applies to standard invoices.
func SnapshotTaxConfigIntoLines(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		line.TaxConfig = billing.MergeTaxConfigs(deps.DefaultTaxConfig, line.TaxConfig)
	}

	return nil
}

// StampInvoicingAppTaxCodes writes the provider tax code that the invoice's invoicing app bills
// into each line's InvoicingAppTaxCode. It runs after SnapshotTaxConfigIntoLines, so a line that
// inherits the invoice default's Tax Code is stamped with that code's mapping.
//
// Every draft calculation restamps all lines, so an edit made while the invoice waits in draft
// (a line change, a new mapping on the Tax Code) is what the draft sync sends to the app. The
// stamp freezes when the invoice leaves draft: nothing recalculates it afterwards, so later Tax
// Code changes no longer reach the invoice.
//
// A line without a Tax Code, or whose Tax Code has no mapping for the invoicing app, gets "":
// the provider's default applies. Gathering invoices are skipped because they are not tied to an
// invoicing app yet. Without an invoicing app type (invoice simulation) lines are left untouched.
func StampInvoicingAppTaxCodes(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering || deps.InvoicingAppType == "" {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		if line.TaxConfig == nil || line.TaxConfig.TaxCodeID == "" {
			line.InvoicingAppTaxCode = lo.ToPtr("")
			continue
		}

		tc, ok := deps.TaxCodes.Get(line.TaxConfig.TaxCodeID)
		if !ok {
			return fmt.Errorf("stamping invoicing app tax code on line [%s]: tax code [%s] is not resolved: %w", line.ID, line.TaxConfig.TaxCodeID, taxcode.NewTaxCodeNotFoundError(line.TaxConfig.TaxCodeID))
		}

		mapping, _ := tc.GetAppMapping(deps.InvoicingAppType)
		line.InvoicingAppTaxCode = lo.ToPtr(mapping.TaxCode)
	}

	return nil
}
