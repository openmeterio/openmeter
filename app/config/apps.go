package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type AppsConfiguration struct {
	// BaseURL is the base URL for the Stripe webhook.
	BaseURL string `yaml:"baseURL"`

	Stripe AppStripeConfiguration `yaml:"stripe"`
}

func (c AppsConfiguration) Validate() error {
	var errs []error

	if c.BaseURL == "" {
		errs = append(errs, errors.New("base URL is required"))
	}

	if err := c.Stripe.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type AppStripeConfiguration struct {
	DisableWebhookRegistration bool   `yaml:"disableWebhookRegistration"`
	WebhookURLPattern          string `yaml:"webhookURLPattern" mapstructure:"webhookURLPattern"`
}

func (c AppStripeConfiguration) Validate() error {
	return nil
}

func ConfigureApps(v *viper.Viper, flags *pflag.FlagSet) {
	v.SetDefault("apps.baseURL", "https://example.com")

	flags.Bool("stripe-disable-webhook-registration", false, "Disable webhook registration for Stripe [for local development]")
	flags.String("stripe-webhook-url-pattern", "", "Webhook URL pattern for Stripe [for local development]")
	_ = v.BindPFlag("apps.stripe.disableWebhookRegistration", flags.Lookup("stripe-disable-webhook-registration"))
	v.SetDefault("apps.stripe.disableWebhookRegistration", false)
	v.SetDefault("apps.stripe.webhookURLPattern", "")
}
