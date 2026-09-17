package config

import (
	"errors"

	"github.com/spf13/viper"
)

type CreditsConfiguration struct {
	Enabled                 bool `yaml:"enabled"`
	EnableCreditThenInvoice bool `yaml:"enableCreditThenInvoice"`
}

func (c CreditsConfiguration) Validate() error {
	var errs []error

	// Add any validation logic for CreditsConfiguration here if needed.

	return errors.Join(errs...)
}

func ConfigureCredits(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
	v.SetDefault(prefixer("enableCreditThenInvoice"), false)
}
