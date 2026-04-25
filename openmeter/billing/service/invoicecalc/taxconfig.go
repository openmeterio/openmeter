package invoicecalc

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// SnapshotTaxConfigIntoLines merges the invoice's DefaultTaxConfig into each line and
// stamps the resolved TaxCode entity (and its ID) onto the line's TaxConfig.
// Skipped for gathering invoices — snapshotting only applies to standard invoices.
func SnapshotTaxConfigIntoLines(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		line.TaxConfig = productcatalog.MergeTaxConfigs(invoice.Workflow.Config.Invoicing.DefaultTaxConfig, line.TaxConfig)

		if line.TaxConfig == nil || line.TaxConfig.Stripe == nil {
			continue
		}

		tc, ok := deps.TaxCodes.Get(line.TaxConfig.Stripe.Code)
		if !ok {
			continue
		}

		if line.TaxConfig.TaxCodeID == nil {
			line.TaxConfig.TaxCodeID = lo.ToPtr(tc.ID)
		}
		line.TaxConfig.TaxCode = tc
	}

	return nil
}
