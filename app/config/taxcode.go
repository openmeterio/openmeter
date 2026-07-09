package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

type TaxCodeConfiguration struct {
	Seeds []TaxCodeSeed `mapstructure:"seeds"`
}

// TaxCodeSeed defines a single tax code entry to provision for every namespace.
// A single seed may carry both DefaultInvoicing and DefaultCreditGrant; the two flags are
// independent and the exactly-one rule is enforced per flag.
type TaxCodeSeed struct {
	Key                string              `mapstructure:"key"`
	Name               string              `mapstructure:"name"`
	Description        *string             `mapstructure:"description"`
	AppMappings        []TaxCodeAppMapping `mapstructure:"appMappings"`
	DefaultInvoicing   bool                `mapstructure:"defaultInvoicing"`
	DefaultCreditGrant bool                `mapstructure:"defaultCreditGrant"`
}

type TaxCodeAppMapping struct {
	AppType string `mapstructure:"appType"`
	TaxCode string `mapstructure:"taxCode"`
}

func (c TaxCodeConfiguration) Validate() error {
	var errs []error

	if len(c.Seeds) == 0 {
		errs = append(errs, errors.New("seeds must not be empty"))
		return errors.Join(errs...)
	}

	keys := make(map[string]struct{}, len(c.Seeds))
	defaultInvoicingCount := 0
	defaultCreditGrantCount := 0

	for i, seed := range c.Seeds {
		trimmedKey := strings.TrimSpace(seed.Key)
		if trimmedKey == "" {
			errs = append(errs, fmt.Errorf("seed[%d]: key must not be empty", i))
		}

		if strings.TrimSpace(seed.Name) == "" {
			errs = append(errs, fmt.Errorf("seed[%d]: name must not be empty", i))
		}

		if trimmedKey != "" {
			if _, exists := keys[trimmedKey]; exists {
				errs = append(errs, fmt.Errorf("seed[%d]: duplicate key %q", i, seed.Key))
			}
			keys[trimmedKey] = struct{}{}
		}

		if seed.DefaultInvoicing {
			defaultInvoicingCount++
		}

		if seed.DefaultCreditGrant {
			defaultCreditGrantCount++
		}

		for j, mapping := range seed.AppMappings {
			trimmedAppType := strings.TrimSpace(mapping.AppType)
			trimmedTaxCode := strings.TrimSpace(mapping.TaxCode)

			if trimmedAppType == "" {
				errs = append(errs, fmt.Errorf("seed[%d].appMappings[%d]: appType must not be empty", i, j))
			}

			if trimmedTaxCode == "" {
				errs = append(errs, fmt.Errorf("seed[%d].appMappings[%d]: taxCode must not be empty", i, j))
			} else if trimmedAppType == "stripe" && !taxcode.TaxCodeStripeRegexp.MatchString(trimmedTaxCode) {
				errs = append(errs, fmt.Errorf("seed[%d].appMappings[%d]: taxCode %q is not a valid Stripe tax code (must match %s)", i, j, trimmedTaxCode, taxcode.TaxCodeStripeRegexp.String()))
			}
		}
	}

	if defaultInvoicingCount != 1 {
		errs = append(errs, fmt.Errorf("exactly one seed must have defaultInvoicing=true, got %d", defaultInvoicingCount))
	}

	if defaultCreditGrantCount != 1 {
		errs = append(errs, fmt.Errorf("exactly one seed must have defaultCreditGrant=true, got %d", defaultCreditGrantCount))
	}

	return errors.Join(errs...)
}

func ConfigureTaxCode(v *viper.Viper) {
	v.SetDefault("taxcode.seeds", []map[string]any{
		{
			"key":              taxcode.ProviderDefaultTaxCodeKey,
			"name":             "Provider default",
			"defaultInvoicing": true,
		},
		{
			"key":                taxcode.CreditGrantTaxCodeKey,
			"name":               "Nontaxable",
			"defaultCreditGrant": true,
			"appMappings": []map[string]any{
				{
					"appType": "stripe",
					"taxCode": "txcd_00000000",
				},
			},
		},
	})
}
