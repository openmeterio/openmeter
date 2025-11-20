package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type COGSConfiguration struct {
	PGPollingInterval time.Duration
}

func (c COGSConfiguration) Validate() error {
	if c.PGPollingInterval < time.Second {
		return errors.New("pg polling interval must be greater than or equal to 1s")
	}

	return nil
}

func ConfigureCOGS(v *viper.Viper) {
	v.SetDefault("cogs.pgPollingInterval", 10*time.Second)
}
