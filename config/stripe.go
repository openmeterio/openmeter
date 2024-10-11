package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// StripeAppConfig is the configuration for Stripe.
type StripeAppConfig struct {
	IncomingWebhook StripeAppIncomingWebhookConfig `yaml:"incomingWebhook"`
}

// Validate validates the configuration.
func (c StripeAppConfig) Validate() error {
	var errs []error

	if err := c.IncomingWebhook.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("incoming webhook: %w", err))
	}

	return errors.Join(errs...)
}

// StripeAppIncomingWebhookConfig is the configuration for the Stripe webhook.
type StripeAppIncomingWebhookConfig struct {
	// BaseURL is the base URL for the Stripe webhook.
	BaseURL string `yaml:"baseURL"`
}

func (c StripeAppIncomingWebhookConfig) Validate() error {
	var errs []error

	if c.BaseURL == "" {
		errs = append(errs, errors.New("base URL is required"))
	}

	return errors.Join(errs...)
}

// ConfigureStripe configures the default values for Stripe.
func ConfigureStripe(v *viper.Viper) {
	v.SetDefault("stripeApp.incomingWebhook.baseURL", "https://example.com")
}
