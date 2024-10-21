package config

import "github.com/spf13/viper"

type AppsConfiguration struct {
	Enabled bool
}

func (c AppsConfiguration) Validate() error {
	return nil
}

func ConfigureApps(v *viper.Viper) {
	v.SetDefault("apps.enabled", false)
}
