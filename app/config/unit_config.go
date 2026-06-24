package config

import (
	"github.com/spf13/viper"
)

type UnitConfigConfiguration struct {
	Enabled bool `yaml:"enabled"`
}

func (c UnitConfigConfiguration) Validate() error {
	// Add any validation logic for UnitConfigConfiguration here if needed.
	return nil
}

func ConfigureUnitConfig(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
}
