package productcatalog

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/samber/lo"
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

// TaxConfig stores the provider specific tax configs.
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

	return NewValidationError(errors.Join(errs...))
}

func MergeTaxConfigs(l, r *TaxConfig) *TaxConfig {
	if l != nil && r != nil {
		return &TaxConfig{
			Behavior: lo.CoalesceOrEmpty(r.Behavior, l.Behavior),
			Stripe:   lo.CoalesceOrEmpty(r.Stripe, l.Stripe),
		}
	}

	if r != nil {
		return r
	}

	return l
}

var StripeProductTaxCodeRegexp = regexp.MustCompile(`^txcd_\d{8}$`)

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
	if s.Code != "" && !StripeProductTaxCodeRegexp.MatchString(s.Code) {
		return NewValidationError(fmt.Errorf("invalid product tax code: %s", s.Code))
	}

	return nil
}
