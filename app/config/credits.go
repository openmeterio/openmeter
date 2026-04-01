package config

import (
	"errors"

	"github.com/spf13/viper"
)

type CreditsConfiguration struct {
	Enabled bool `yaml:"enabled"`
}

func (c CreditsConfiguration) Validate() error {
	var errs []error

	// Add any validation logic for CreditsConfiguration here if needed.

	return errors.Join(errs...)
}

func ConfigureCredits(v *viper.Viper) {
	v.SetDefault("credits.enabled", false)
}
