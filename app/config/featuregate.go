package config

import (
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/featuregate"
)

type FeatureGateConfiguration struct {
	Enabled bool              `yaml:"enabled"`
	Flags   featuregate.Flags `yaml:"flags"`
}

func ConfigureFeatureGate(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("enabled"), false)
	v.SetDefault(prefixer("flags"), make(featuregate.Flags))
}
