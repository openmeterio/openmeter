package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CreditConfiguration struct {
	Enabled bool `yaml:"enabled"`
}

func (c CreditConfiguration) Validate() error {
	var errs []error

	return errors.Join(errs...)
}

func ConfigureCredit(v *viper.Viper, flags *pflag.FlagSet) {
	v.SetDefault("credit.enabled", false)
	flags.Bool("credit-enabled", false, "Enable credit system")
	_ = v.BindPFlag("credit.enabled", flags.Lookup("credit-enabled"))
}
