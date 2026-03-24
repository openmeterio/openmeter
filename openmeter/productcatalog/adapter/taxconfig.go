package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// BackfillTaxConfig fills in missing legacy TaxConfig fields from the new tax_behavior column
// and the TaxCode entity's app mappings.
func BackfillTaxConfig(cfg *productcatalog.TaxConfig, taxBehavior *productcatalog.TaxBehavior, tc *taxcode.TaxCode) *productcatalog.TaxConfig {
	// Resolve Stripe code from TaxCode app mappings.
	var stripeCode string
	if tc != nil {
		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			stripeCode = m.TaxCode
		}
	}

	// Nothing to backfill.
	if taxBehavior == nil && stripeCode == "" {
		return cfg
	}

	if cfg == nil {
		cfg = &productcatalog.TaxConfig{}
	}

	if cfg.Behavior == nil && taxBehavior != nil {
		cfg.Behavior = taxBehavior
	}

	if cfg.Stripe == nil && stripeCode != "" {
		cfg.Stripe = &productcatalog.StripeTaxConfig{
			Code: stripeCode,
		}
	}

	if cfg.TaxCodeID == nil && tc != nil && tc.ID != "" {
		cfg.TaxCodeID = &tc.ID
	}

	return cfg
}
