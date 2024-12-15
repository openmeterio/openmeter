package config

import (
	"errors"

	"github.com/spf13/viper"
)

type AppsConfiguration struct {
	Enabled bool
	// BaseURL is the base URL for the Stripe webhook.
	BaseURL string `yaml:"baseURL"`
}

func (c AppsConfiguration) Validate() error {
	var errs []error

	if !c.Enabled {
		return nil
	}

	if c.BaseURL == "" {
		errs = append(errs, errors.New("base URL is required"))
	}

	return errors.Join(errs...)
}

func ConfigureApps(v *viper.Viper) {
	v.SetDefault("apps.enabled", false)
	v.SetDefault("apps.baseURL", "https://example.com")
}
