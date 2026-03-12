package invoicecalc

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// SnapshotTaxConfigIntoLines takes the tax configuration from the invoice's workflow and adds it to the lines.
// It also resolves TaxCodeId references to full TaxCode entities using the pre-fetched tax codes from dependencies.
func SnapshotTaxConfigIntoLines(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		line.TaxConfig = productcatalog.MergeTaxConfigs(invoice.Workflow.Config.Invoicing.DefaultTaxConfig, line.TaxConfig)

		// Resolve TaxCodeId to a full TaxCode entity
		if line.TaxConfig != nil && line.TaxConfig.TaxCodeId != nil && deps.TaxCodes != nil {
			if tc, ok := deps.TaxCodes[*line.TaxConfig.TaxCodeId]; ok {
				line.StandardLineBase.TaxCode = &tc
			}
		}
	}

	return nil
}
