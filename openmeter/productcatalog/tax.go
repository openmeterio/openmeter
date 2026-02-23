package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

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
	Behavior *TaxBehavior     `json:"behavior,omitempty"`
	Stripe   *StripeTaxConfig `json:"stripe,omitempty"`
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
	out := TaxConfig{
		Behavior: c.Behavior,
	}

	if c.Stripe != nil {
		out.Stripe = lo.ToPtr(c.Stripe.Clone())
	}

	return out
}

func MergeTaxConfigs(base, overrides *TaxConfig) *TaxConfig {
	if base != nil && overrides != nil {
		return &TaxConfig{
			Behavior: lo.CoalesceOrEmpty(overrides.Behavior, base.Behavior),
			Stripe:   lo.CoalesceOrEmpty(overrides.Stripe, base.Stripe),
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
