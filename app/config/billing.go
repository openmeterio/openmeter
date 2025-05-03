package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type BillingConfiguration struct {
	AdvancementStrategy billing.AdvancementStrategy
	Worker              BillingWorkerConfiguration
	FeatureSwitches     BillingFeatureSwitchesConfiguration
}

func (c BillingConfiguration) Validate() error {
	var errs []error
	if err := c.Worker.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := c.AdvancementStrategy.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := c.FeatureSwitches.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type BillingFeatureSwitchesConfiguration struct {
	NamespaceLockdown []string
}

func (c BillingFeatureSwitchesConfiguration) Validate() error {
	return nil
}

func ConfigureBilling(v *viper.Viper, flags *pflag.FlagSet) {
	ConfigureBillingWorker(v)

	// Allow overriding the advancement strategy for local development purposes where we are
	// likely to use the same configuration file for multiple services.
	flags.String("billing-advancement-strategy", "foreground", "Advancement strategy for billing")
	_ = v.BindPFlag("billing.advancementStrategy", flags.Lookup("billing-advancement-strategy"))
	v.SetDefault("billing.advancementStrategy", billing.ForegroundAdvancementStrategy)
}
