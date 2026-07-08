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

// MergeTaxConfigs merges two TaxConfigs with overrides taking precedence.
//
// Stripe and TaxCodeID are two encodings of the same intent-level tax-code identity, so they
// merge as a unit: a config that overrides only the Stripe code must not inherit the base's
// (different) TaxCodeID, which would leave the result pointing at two different tax entities.
func MergeTaxConfigs(base, overrides *TaxConfig) *TaxConfig {
	if base != nil && overrides != nil {
		stripe, taxCodeID := base.Stripe, base.TaxCodeID
		if overrides.Stripe != nil || overrides.TaxCodeID != nil {
			stripe, taxCodeID = overrides.Stripe, overrides.TaxCodeID
		}

		return &TaxConfig{
			Behavior:  lo.CoalesceOrEmpty(overrides.Behavior, base.Behavior),
			Stripe:    stripe,
			TaxCodeID: taxCodeID,
		}
	}

	if overrides != nil {
		c := overrides.Clone()
		return &c
	}

	if base != nil {
		c := base.Clone()
		return &c
	}

	return nil
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
	TaxCodeID string       `json:"tax_code_id"`
}

func (c TaxCodeConfig) Validate() error {
	var errs []error

	if c.Behavior != nil {
		if err := c.Behavior.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.TaxCodeID == "" {
		errs = append(errs, fmt.Errorf("tax code id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

// ResolveTaxConfigInput describes a TaxConfig resolution request.
type ResolveTaxConfigInput struct {
	Namespace string
	// Cfg is mutated in place: TaxCodeID and Stripe are cross-populated onto it.
	Cfg *TaxConfig
	// IncludeDeleted allows resolving a soft-deleted TaxCode by ID instead of rejecting it.
	// Defaults to false (reject): most callers are accepting a reference that must remain live
	// going forward (plan/addon rate cards, subscription items, billing profile/customer-override
	// defaults being newly set). Set true only for continuity reads that re-derive an
	// already-persisted, possibly-since-deleted reference (billing invoice/gathering-line
	// snapshotting).
	IncludeDeleted bool
}

// ResolveTaxConfig cross-populates TaxCodeID and provider-specific codes on the pointed-to
// config so the persisted record is internally consistent. Four input cases:
//   - Only TaxCodeID, or both TaxCodeID and Stripe.Code (same branch below): looks up the
//     entity by ID and validates it exists (400 if not). Rejects a soft-deleted entity (400)
//     unless input.IncludeDeleted is true. Sets Stripe from the entity's Stripe app mapping (or
//     clears Stripe if the entity has no mapping). When both are supplied, TaxCodeID wins; the
//     caller-supplied Stripe.Code is discarded.
//   - Only Stripe.Code: upserts the TaxCode entity via GetOrCreateByAppMapping and stamps
//     TaxCodeID (idempotent; updating the code txcd_A → txcd_B updates the FK). IncludeDeleted is
//     irrelevant here: GetTaxCodeByAppMapping already filters soft-deleted rows, and the
//     (namespace, key) unique index only applies to non-deleted rows, so re-migrating a Stripe
//     code whose prior TaxCode was soft-deleted always creates a fresh entity.
//   - Neither: no-op.
//
// No-op when input.Cfg is nil.
func ResolveTaxConfig(ctx context.Context, svc taxcode.Service, input ResolveTaxConfigInput) error {
	cfg := input.Cfg
	if cfg == nil {
		return nil
	}

	if svc == nil {
		return fmt.Errorf("taxcode service is required")
	}

	switch {
	case cfg.TaxCodeID != nil:
		tc, err := svc.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: input.Namespace, ID: *cfg.TaxCodeID},
		})
		if err != nil {
			if taxcode.IsTaxCodeNotFoundError(err) {
				return models.NewGenericValidationError(fmt.Errorf("tax code %s not found", *cfg.TaxCodeID))
			}
			return fmt.Errorf("resolving tax code %s: %w", *cfg.TaxCodeID, err)
		}

		if !input.IncludeDeleted && tc.IsDeleted() {
			return models.NewGenericValidationError(fmt.Errorf("tax code %s not found", *cfg.TaxCodeID))
		}

		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			cfg.Stripe = &StripeTaxConfig{Code: m.TaxCode}
		} else {
			cfg.Stripe = nil
		}

	case cfg.Stripe != nil && cfg.Stripe.Code != "":
		tc, err := svc.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
			Namespace: input.Namespace,
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
		b := *taxBehavior
		cfg.Behavior = &b
	}

	if cfg.Stripe == nil && stripeCode != "" {
		cfg.Stripe = &StripeTaxConfig{Code: stripeCode}
	}

	if cfg.TaxCodeID == nil && tc != nil && tc.ID != "" {
		id := tc.ID
		cfg.TaxCodeID = &id
	}

	return cfg
}
