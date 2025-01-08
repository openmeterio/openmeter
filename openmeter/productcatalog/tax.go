package productcatalog

import (
	"errors"
	"fmt"
	"regexp"
)

// TaxConfig stores the provider specific tax configs.
type TaxConfig struct {
	Stripe *StripeTaxConfig `json:"stripe,omitempty"`
}

func (c *TaxConfig) Equal(v *TaxConfig) bool {
	if c == nil && v == nil {
		return true
	}

	if c == nil || v == nil {
		return false
	}

	return c.Stripe.Equal(v.Stripe)
}

func (c *TaxConfig) Validate() error {
	var errs []error

	if c.Stripe != nil {
		if err := c.Stripe.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid stripe config: %w", err))
		}
	}

	return NewValidationError(errors.Join(errs...))
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
