package invoicecalc

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// SnapshotTaxConfigIntoLines merges the invoice's default tax config (deps.DefaultTaxConfig) into each line and
// stamps the resolved TaxCode entity (and its ID) onto the line's TaxConfig. Lines referencing the
// default get its Tax Code by ID; others are resolved by Stripe code. A line only ever carries the
// Tax Code its ID points at.
// Skipped for gathering invoices — snapshotting only applies to standard invoices.
func SnapshotTaxConfigIntoLines(invoice *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error {
	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	for _, line := range invoice.Lines.OrEmpty() {
		line.TaxConfig = billing.MergeTaxConfigs(deps.DefaultTaxConfig, line.TaxConfig)

		cfg := line.TaxConfig
		if cfg == nil {
			continue
		}

		if cfg.TaxCode != nil && (cfg.TaxCodeID == nil || cfg.TaxCode.ID != *cfg.TaxCodeID) {
			cfg.TaxCode = nil
		}

		if deps.DefaultTaxCode != nil && cfg.TaxCodeID != nil && *cfg.TaxCodeID == deps.DefaultTaxCode.ID {
			cfg.TaxCode = lo.ToPtr(*deps.DefaultTaxCode)

			continue
		}

		if cfg.Stripe == nil {
			continue
		}

		tc, ok := deps.TaxCodes.Get(cfg.Stripe.Code)
		if !ok {
			continue
		}

		if cfg.TaxCodeID == nil {
			cfg.TaxCodeID = lo.ToPtr(tc.ID)
		}

		// Same rule as the adapter on read: a line only carries the Tax Code it references.
		if *cfg.TaxCodeID == tc.ID {
			cfg.TaxCode = tc
		}
	}

	return nil
}
