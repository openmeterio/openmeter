package plan

import (
	"errors"
	"fmt"
	"regexp"
)

// TaxConfig stores the provider specific tax configs.
type TaxConfig struct {
	Stripe *StripeTaxConfig `json:"stripe,omitempty"`
}

func (c TaxConfig) Validate() error {
	var errs []error

	if c.Stripe != nil {
		if err := c.Stripe.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid stripe config: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var StripeProductTaxCodeRegexp = regexp.MustCompile(`^txcd_\d{8}$`)

type StripeTaxConfig struct {
	// Code stores the product tax code.
	// See: https://docs.stripe.com/tax/tax-codes
	// Example:"txcd_10000000"
	Code string `json:"code"`
}

func (s StripeTaxConfig) Validate() error {
	if s.Code != "" && !StripeProductTaxCodeRegexp.MatchString(s.Code) {
		return fmt.Errorf("invalid product tax code: %s", s.Code)
	}

	return nil
}
