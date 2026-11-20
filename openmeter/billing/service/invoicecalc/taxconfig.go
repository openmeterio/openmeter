package invoicecalc

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
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
