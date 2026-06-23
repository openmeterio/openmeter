package config

import (
	"errors"

	"github.com/spf13/viper"
)

type UnitConfigConfiguration struct {
	Enabled bool `yaml:"enabled"`
}

func (c UnitConfigConfiguration) Validate() error {
	var errs []error

	return errors.Join(errs...)
}

func ConfigureUnitConfig(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
}
