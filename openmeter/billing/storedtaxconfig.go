package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// StoredLineTaxConfig is the persisted line copy of a legacy tax config, kept only so invoice
// history stays byte-identical for older readers and the tax_code_id/tax_behavior consistency
// CHECK constraints hold. The domain never reads it: lines carry productcatalog.TaxCodeConfig,
// and renderers rebuild stripe.code and tax_code from the referenced Tax Code on the fly.
type StoredLineTaxConfig struct {
	Behavior  *productcatalog.TaxBehavior     `json:"behavior,omitempty"`
	Stripe    *productcatalog.StripeTaxConfig `json:"stripe,omitempty"`
	TaxCodeID *string                         `json:"tax_code_id,omitempty"`
	TaxCode   *taxcode.TaxCode                `json:"tax_code,omitempty"`
}

// The legacy JSON shape is kept for stored history and event consumers. Standard lines embed the
// referenced Tax Code; gathering lines never carried it.

// NewStoredStandardLineTaxConfig builds the legacy tax config of a standard line, including the
// full Tax Code. A nil config yields nil.
func NewStoredStandardLineTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes map[string]*taxcode.TaxCode) (*StoredLineTaxConfig, error) {
	return newStoredLineTaxConfig(cfg, taxCodes, true)
}

// NewStoredGatheringLineTaxConfig builds the legacy tax config of a gathering line, which never
// embeds the Tax Code. A nil config yields nil.
func NewStoredGatheringLineTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes map[string]*taxcode.TaxCode) (*StoredLineTaxConfig, error) {
	return newStoredLineTaxConfig(cfg, taxCodes, false)
}

func newStoredLineTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes map[string]*taxcode.TaxCode, withTaxCode bool) (*StoredLineTaxConfig, error) {
	if cfg == nil {
		return nil, nil
	}

	out := &StoredLineTaxConfig{Behavior: cfg.Behavior}

	if cfg.TaxCodeID == "" {
		return out, nil
	}

	id := cfg.TaxCodeID
	out.TaxCodeID = &id

	tc, ok := taxCodes[id]
	if !ok || tc == nil {
		return nil, fmt.Errorf("tax code %q not found", id)
	}

	if mapping, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
		out.Stripe = &productcatalog.StripeTaxConfig{Code: mapping.TaxCode}
	}

	if withTaxCode {
		out.TaxCode = tc
	}

	return out, nil
}
