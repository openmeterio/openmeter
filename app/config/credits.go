package config

import (
	"errors"

	"github.com/spf13/viper"
)

type CreditsConfiguration struct {
	Enabled                 bool `yaml:"enabled" mapstructure:"enabled"`
	EnableCreditThenInvoice bool `yaml:"enable_credit_then_invoice" mapstructure:"enable_credit_then_invoice"`
}

func (c CreditsConfiguration) Validate() error {
	var errs []error

	// Add any validation logic for CreditsConfiguration here if needed.

	return errors.Join(errs...)
}

func ConfigureCredits(v *viper.Viper) {
	v.SetDefault("credits.enabled", false)
	v.SetDefault("credits.enable_credit_then_invoice", false)
}
