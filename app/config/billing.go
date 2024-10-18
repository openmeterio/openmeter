package config

import "github.com/spf13/viper"

type BillingConfiguration struct {
	Enabled bool
}

func (c BillingConfiguration) Validate() error {
	return nil
}

func ConfigureBilling(v *viper.Viper) {
	v.SetDefault("billing.enabled", false)
}
