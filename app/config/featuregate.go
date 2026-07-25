package config

import (
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/featuregate"
)

type FeatureGateConfiguration struct {
	Enabled bool              `yaml:"enabled"`
	Flags   featuregate.Flags `yaml:"flags"`
}

func (c FeatureGateConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}
	return c.Flags.Validate()
}

func ConfigureFeatureGate(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
	v.SetDefault(prefixer("flags"), make(featuregate.Flags))
}
