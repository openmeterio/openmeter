package config

import (
	"errors"

	"github.com/spf13/viper"
)

type CreditsConfiguration struct {
	Enabled                 bool `yaml:"enabled"`
	CustomCurrency          bool `yaml:"customCurrency"`
	EnableCreditThenInvoice bool `yaml:"enableCreditThenInvoice"`
}

// IsCustomCurrencyEnabled reports whether plans and add-ons can use custom currencies.
func (c CreditsConfiguration) IsCustomCurrencyEnabled() bool {
	return c.Enabled && c.CustomCurrency
}

func (c CreditsConfiguration) Validate() error {
	var errs []error

	// Add any validation logic for CreditsConfiguration here if needed.

	return errors.Join(errs...)
}

func ConfigureCredits(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
	v.SetDefault(prefixer("customCurrency"), false)
	v.SetDefault(prefixer("enableCreditThenInvoice"), false)
}
