package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

type TaxBehavior string

const (
	InclusiveTaxBehavior TaxBehavior = "inclusive"
	ExclusiveTaxBehavior TaxBehavior = "exclusive"
)

func (t TaxBehavior) Values() []string {
	return []string{
		string(InclusiveTaxBehavior),
		string(ExclusiveTaxBehavior),
	}
}

func (t TaxBehavior) Validate() error {
	if !lo.Contains(t.Values(), string(t)) {
		return fmt.Errorf("invalid tax behavior: %s", t)
	}

	return nil
}

// TaxConfig stores the provider-specific tax configs.
type TaxConfig struct {
	Behavior  *TaxBehavior     `json:"behavior,omitempty"`
	Stripe    *StripeTaxConfig `json:"stripe,omitempty"`
	TaxCodeID *string          `json:"tax_code_id,omitempty"`
}

func (c *TaxConfig) Equal(v *TaxConfig) bool {
	if c == nil && v == nil {
		return true
	}

	if c == nil || v == nil {
		return false
	}

	if (c.Behavior != nil && v.Behavior == nil) || (c.Behavior == nil && v.Behavior != nil) {
		return false
	}

	if c.Behavior != nil && *c.Behavior != *v.Behavior {
		return false
	}

	if (c.TaxCodeID != nil && v.TaxCodeID == nil) || (c.TaxCodeID == nil && v.TaxCodeID != nil) {
		return false
	}

	if c.TaxCodeID != nil && *c.TaxCodeID != *v.TaxCodeID {
		return false
	}

	return c.Stripe.Equal(v.Stripe)
}

func (c *TaxConfig) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if c.Behavior != nil {
		if err := c.Behavior.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.Stripe != nil {
		if err := c.Stripe.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid stripe config: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c TaxConfig) Clone() TaxConfig {
	out := TaxConfig{}

	if c.Behavior != nil {
		out.Behavior = lo.ToPtr(*c.Behavior)
	}

	if c.Stripe != nil {
		out.Stripe = lo.ToPtr(c.Stripe.Clone())
	}

	if c.TaxCodeID != nil {
		out.TaxCodeID = lo.ToPtr(*c.TaxCodeID)
	}

	return out
}

type StripeTaxConfig struct {
	// Code stores the product tax code.
	// See: https://docs.stripe.com/tax/tax-codes
	// Example:"txcd_10000000"
	Code string `json:"code"`
}

func (s *StripeTaxConfig) Equal(v *StripeTaxConfig) bool {
	if s == nil && v == nil {
		return true
	}

	if s == nil || v == nil {
		return false
	}

	return s.Code == v.Code
}

func (s *StripeTaxConfig) Validate() error {
	if s.Code != "" && !taxcode.TaxCodeStripeRegexp.MatchString(s.Code) {
		return models.NewGenericValidationError(fmt.Errorf("invalid product tax code: %s", s.Code))
	}

	return nil
}

func (s StripeTaxConfig) Clone() StripeTaxConfig {
	return s
}

// TaxCodeConfig is the internal tax reference: a Tax Code ID plus an optional behavior.
//
// The omitempty on TaxCodeID matters: an empty-ID config must serialize as {}, not
// {"tax_code_id": ""} — the DB CHECK constraint compares the JSON against the column with
// IS NOT DISTINCT FROM, and legacy TaxConfig readers decode this document.
type TaxCodeConfig struct {
	Behavior  *TaxBehavior `json:"behavior,omitempty"`
	TaxCodeID string       `json:"tax_code_id,omitempty"`
}

// IsEmpty reports whether the config carries neither a Tax Code nor a behavior.
func (c TaxCodeConfig) IsEmpty() bool {
	return c.TaxCodeID == "" && c.Behavior == nil
}

func (c TaxCodeConfig) Validate() error {
	if c.Behavior != nil {
		return models.NewNillableGenericValidationError(c.Behavior.Validate())
	}

	return nil
}

// Equal treats nil and the zero value as the same "no tax config": adapters produce either one.
func (c *TaxCodeConfig) Equal(v *TaxCodeConfig) bool {
	left, right := lo.FromPtr(c), lo.FromPtr(v)

	if left.TaxCodeID != right.TaxCodeID {
		return false
	}

	if (left.Behavior == nil) != (right.Behavior == nil) {
		return false
	}

	return left.Behavior == nil || *left.Behavior == *right.Behavior
}

func (c TaxCodeConfig) Clone() TaxCodeConfig {
	out := TaxCodeConfig{TaxCodeID: c.TaxCodeID}

	if c.Behavior != nil {
		out.Behavior = lo.ToPtr(*c.Behavior)
	}

	return out
}

// ToTaxConfig converts TaxCodeConfig to TaxConfig (without provider-specific fields).
func (c TaxCodeConfig) ToTaxConfig() TaxConfig {
	out := TaxConfig{}
	if c.Behavior != nil {
		out.Behavior = lo.ToPtr(*c.Behavior)
	}

	if c.TaxCodeID != "" {
		out.TaxCodeID = lo.ToPtr(c.TaxCodeID)
	}

	return out
}

// ToLegacyTaxConfig renders the reference in the legacy v1 shape, taking stripe.code from the
// referenced Tax Code's current Stripe mapping. stripe.code is left out when the Tax Code isn't in
// taxCodes or has no Stripe mapping. Only for code that still has to speak the legacy shape.
func (c TaxCodeConfig) ToLegacyTaxConfig(taxCodes map[string]*taxcode.TaxCode) TaxConfig {
	out := c.ToTaxConfig()

	if tc := taxCodes[c.TaxCodeID]; tc != nil && c.TaxCodeID != "" {
		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			out.Stripe = &StripeTaxConfig{Code: m.TaxCode}
		}
	}

	return out
}

// TaxCodeConfigFrom extracts the lean reference fields from a full TaxConfig.
// Returns the zero value when cfg is nil or when neither Behavior nor TaxCodeID is set
// (e.g. Stripe-only config).
func TaxCodeConfigFrom(cfg *TaxConfig) TaxCodeConfig {
	if cfg == nil {
		return TaxCodeConfig{}
	}

	out := TaxCodeConfig{}
	if cfg.Behavior != nil {
		out.Behavior = lo.ToPtr(*cfg.Behavior)
	}

	if cfg.TaxCodeID != nil {
		out.TaxCodeID = *cfg.TaxCodeID
	}

	return out
}

// FromDBTaxColumns reads tax settings from the tax_code_id and tax_behavior columns only. The legacy
// tax_config JSON may still hold a Stripe code on old rows, but those rows were backfilled with a
// tax_code_id (a CHECK constraint keeps it that way), so the columns are the source of truth.
func FromDBTaxColumns(taxCodeID *string, behavior *TaxBehavior) *TaxCodeConfig {
	if taxCodeID == nil && behavior == nil {
		return nil
	}

	return &TaxCodeConfig{
		TaxCodeID: lo.FromPtr(taxCodeID),
		Behavior:  behavior,
	}
}
