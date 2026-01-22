package invoicecalc

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// SnapshotTaxConfigIntoLines takes the tax configuration from the invoice's workflow and adds it to the lines.
func SnapshotTaxConfigIntoLines(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		line.TaxConfig = productcatalog.MergeTaxConfigs(invoice.Workflow.Config.Invoicing.DefaultTaxConfig, line.TaxConfig)
	}

	return nil
}
