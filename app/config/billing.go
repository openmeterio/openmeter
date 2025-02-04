package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type BillingConfiguration struct {
	Enabled             bool
	AdvancementStrategy billing.AdvancementStrategy
	Worker              BillingWorkerConfiguration
}

func (c BillingConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error
	if err := c.Worker.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := c.AdvancementStrategy.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func ConfigureBilling(v *viper.Viper, flags *pflag.FlagSet) {
	v.SetDefault("billing.enabled", false)

	ConfigureBillingWorker(v)

	// Allow overriding the advancement strategy for local development purposes where we are
	// likely to use the same configuration file for multiple services.
	flags.String("billing-advancement-strategy", "foreground", "Advancement strategy for billing")
	_ = v.BindPFlag("billing.advancementStrategy", flags.Lookup("billing-advancement-strategy"))
	v.SetDefault("billing.advancementStrategy", billing.ForegroundAdvancementStrategy)
}
