package config

import (
	"github.com/spf13/viper"
)

type SinkConfiguration struct {
}

func (c SinkConfiguration) Validate() error {
	return nil
}

// Configure configures some defaults in the Viper instance.
func configureSink(v *viper.Viper) {
}
