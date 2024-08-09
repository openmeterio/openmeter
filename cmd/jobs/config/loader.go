package config

import (
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/config"
)

func LoadConfig(fileName string) (config.Configuration, error) {
	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	config.SetViperDefaults(v, flags)
	if fileName != "" {
		v.SetConfigFile(fileName)
	}

	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return config.Configuration{}, err
	}

	var conf config.Configuration
	err = v.Unmarshal(&conf)
	if err != nil {
		return conf, err
	}

	return conf, conf.Validate()
}

var defaultConfig *config.Configuration

func GetConfig() (config.Configuration, error) {
	if defaultConfig == nil {
		return config.Configuration{}, errors.New("config not set")
	}

	return *defaultConfig, nil
}

func SetConfig(c config.Configuration) {
	defaultConfig = &c
}
