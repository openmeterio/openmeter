package config

import (
	"errors"

	"github.com/spf13/viper"
)

// StripeConfig is the configuration for Stripe.
type StripeConfig struct {
	Webhook StripeWebhookConfig `yaml:"webhook"`
}

// Validate validates the configuration.
func (c StripeConfig) Validate() error {
	var errs []error

	if err := c.Webhook.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// StripeWebhookConfig is the configuration for the Stripe webhook.
type StripeWebhookConfig struct {
	// BaseURL is the base URL for the Stripe webhook.
	BaseURL string `yaml:"baseURL"`
}

func (c StripeWebhookConfig) Validate() error {
	var errs []error

	if c.BaseURL == "" {
		errs = append(errs, errors.New("webhook base URL is required"))
	}

	return errors.Join(errs...)
}

// ConfigureStripe configures the default values for Stripe.
func ConfigureStripe(v *viper.Viper) {
	v.SetDefault("stripe.webhook.baseURL", "https://example.com")
}
