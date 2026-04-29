package productcatalog

import (
	"context"
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
	// TaxCode is the resolved TaxCode entity, stamped at invoice snapshot time.
	// Present only on invoice lines (persisted in JSONB); nil on profile/rate-card configs.
	TaxCode *taxcode.TaxCode `json:"tax_code,omitempty"`
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

	if c.TaxCode != nil {
		tc := *c.TaxCode
		tc.AppMappings = append(taxcode.TaxCodeAppMappings(nil), c.TaxCode.AppMappings...)
		if c.TaxCode.Description != nil {
			tc.Description = lo.ToPtr(*c.TaxCode.Description)
		}
		out.TaxCode = &tc
	}

	return out
}

func MergeTaxConfigs(base, overrides *TaxConfig) *TaxConfig {
	if base != nil && overrides != nil {
		return &TaxConfig{
			Behavior:  lo.CoalesceOrEmpty(overrides.Behavior, base.Behavior),
			Stripe:    lo.CoalesceOrEmpty(overrides.Stripe, base.Stripe),
			TaxCodeID: lo.CoalesceOrEmpty(overrides.TaxCodeID, base.TaxCodeID),
		}
	}

	if overrides != nil {
		return overrides
	}

	return base
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

// TaxCodeConfig holds a lean reference to a tax code entry — only the FK and the behavior flag.
// Used in charge intents where provider-specific fields (e.g. Stripe.Code) are not stored and
// are resolved at invoice snapshot time via BackfillTaxConfig.
type TaxCodeConfig struct {
	Behavior  *TaxBehavior `json:"behavior,omitempty"`
	TaxCodeID *string      `json:"tax_code_id,omitempty"`
}

func (c *TaxCodeConfig) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if c.Behavior != nil {
		if err := c.Behavior.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ToTaxConfig converts TaxCodeConfig to TaxConfig (without provider-specific fields).
func (c *TaxCodeConfig) ToTaxConfig() *TaxConfig {
	if c == nil {
		return nil
	}

	return &TaxConfig{
		Behavior:  c.Behavior,
		TaxCodeID: c.TaxCodeID,
	}
}

// TaxCodeConfigFrom extracts the lean reference fields from a full TaxConfig.
func TaxCodeConfigFrom(cfg *TaxConfig) *TaxCodeConfig {
	if cfg == nil {
		return nil
	}

	return &TaxCodeConfig{
		Behavior:  cfg.Behavior,
		TaxCodeID: cfg.TaxCodeID,
	}
}

// ResolveTaxConfig cross-populates TaxCodeID and provider-specific codes on the pointed-to
// config so the persisted record is internally consistent. Four input cases:
//   - Only TaxCodeID: looks up the entity, validates it exists (400 if not), and sets Stripe
//     from the entity's Stripe app mapping (or clears Stripe if the entity has no mapping).
//   - Only Stripe.Code: upserts the TaxCode entity via GetOrCreateByAppMapping and stamps
//     TaxCodeID (idempotent; updating the code txcd_A → txcd_B updates the FK).
//   - Both TaxCodeID and Stripe.Code: TaxCodeID wins. Stripe is overridden from the entity's
//     Stripe app mapping (or cleared if the entity has no mapping); the caller-supplied
//     Stripe.Code is discarded.
//   - Neither: no-op.
//
// No-op when cfg is nil.
func ResolveTaxConfig(ctx context.Context, svc taxcode.Service, namespace string, cfg *TaxConfig) error {
	if cfg == nil {
		return nil
	}

	switch {
	case cfg.TaxCodeID != nil:
		tc, err := svc.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: *cfg.TaxCodeID},
		})
		if err != nil {
			if taxcode.IsTaxCodeNotFoundError(err) {
				return models.NewGenericValidationError(fmt.Errorf("tax code %s not found", *cfg.TaxCodeID))
			}
			return fmt.Errorf("resolving tax code %s: %w", *cfg.TaxCodeID, err)
		}
		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			cfg.Stripe = &StripeTaxConfig{Code: m.TaxCode}
		} else {
			cfg.Stripe = nil
		}

	case cfg.Stripe != nil && cfg.Stripe.Code != "":
		tc, err := svc.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
			Namespace: namespace,
			AppType:   app.AppTypeStripe,
			TaxCode:   cfg.Stripe.Code,
		})
		if err != nil {
			return fmt.Errorf("resolving tax code for stripe code %s: %w", cfg.Stripe.Code, err)
		}
		cfg.TaxCodeID = lo.ToPtr(tc.ID)
	}

	return nil
}

// BackfillTaxConfig fills in missing legacy TaxConfig fields from the new tax_behavior column
// and the TaxCode entity's app mappings.
func BackfillTaxConfig(cfg *TaxConfig, taxBehavior *TaxBehavior, tc *taxcode.TaxCode) *TaxConfig {
	var stripeCode string
	if tc != nil {
		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			stripeCode = m.TaxCode
		}
	}

	if taxBehavior == nil && stripeCode == "" {
		return cfg
	}

	if cfg == nil {
		cfg = &TaxConfig{}
	}

	if cfg.Behavior == nil && taxBehavior != nil {
		cfg.Behavior = taxBehavior
	}

	if cfg.Stripe == nil && stripeCode != "" {
		cfg.Stripe = &StripeTaxConfig{Code: stripeCode}
	}

	if cfg.TaxCodeID == nil && tc != nil && tc.ID != "" {
		cfg.TaxCodeID = &tc.ID
	}

	return cfg
}
