package internal

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
)

func loadConfig(configFile string) error {
	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	config.SetViperDefaults(v, flags)
	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return err
	}

	err = v.Unmarshal(&Config)
	if err != nil {
		return err
	}

	return Config.Validate()
}
