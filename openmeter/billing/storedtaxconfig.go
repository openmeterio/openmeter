package billing

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clone"
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
func NewStoredStandardLineTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes productcatalog.TaxCodesByID) (*StoredLineTaxConfig, error) {
	return newStoredLineTaxConfig(newStoredLineTaxConfigInput{
		Config:         cfg,
		TaxCodes:       taxCodes,
		IncludeTaxCode: true,
	})
}

// NewStoredGatheringLineTaxConfig builds the legacy tax config of a gathering line, which never
// embeds the Tax Code. A nil config yields nil.
func NewStoredGatheringLineTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes productcatalog.TaxCodesByID) (*StoredLineTaxConfig, error) {
	return newStoredLineTaxConfig(newStoredLineTaxConfigInput{
		Config:   cfg,
		TaxCodes: taxCodes,
	})
}

type newStoredLineTaxConfigInput struct {
	Config   *productcatalog.TaxCodeConfig
	TaxCodes productcatalog.TaxCodesByID

	// IncludeTaxCode embeds the full Tax Code entity in the stored JSON. Standard lines did in the
	// legacy shape; gathering lines didn't.
	IncludeTaxCode bool
}

func newStoredLineTaxConfig(in newStoredLineTaxConfigInput) (*StoredLineTaxConfig, error) {
	if in.Config == nil {
		return nil, nil
	}

	out := newStoredLineTaxConfigWithoutTaxCode(in.Config)

	if in.Config.TaxCodeID == "" {
		return out, nil
	}

	tc, ok := in.TaxCodes.Get(in.Config.TaxCodeID)
	if !ok {
		return nil, fmt.Errorf("tax code %q not found", in.Config.TaxCodeID)
	}

	if mapping, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
		out.Stripe = &productcatalog.StripeTaxConfig{Code: mapping.TaxCode}
	}

	if in.IncludeTaxCode {
		out.TaxCode = tc
	}

	return out, nil
}

// newStoredLineTaxConfigWithoutTaxCode copies only the behavior and the Tax Code ID, so it needs
// no Tax Code lookup.
func newStoredLineTaxConfigWithoutTaxCode(cfg *productcatalog.TaxCodeConfig) *StoredLineTaxConfig {
	if cfg == nil {
		return nil
	}

	// Clone the behavior so the stored copy doesn't share its pointer with the line.
	out := &StoredLineTaxConfig{Behavior: clone.CloneEnumPtr(cfg.Behavior)}

	if cfg.TaxCodeID != "" {
		out.TaxCodeID = lo.ToPtr(cfg.TaxCodeID)
	}

	return out
}
