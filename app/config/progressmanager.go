package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/redis"
)

// ProgressManagerConfiguration stores the configuration parameters for the progress manager
type ProgressManagerConfiguration struct {
	Enabled    bool
	KeyPrefix  string
	Expiration time.Duration
	Redis      redis.Config
}

// Validate checks if the configuration is valid
func (c ProgressManagerConfiguration) Validate() error {
	var errs []error

	if !c.Enabled {
		return nil
	}

	if c.Expiration <= 0 {
		errs = append(errs, errors.New("expiration must be greater than 0"))
	}

	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "redis"))
	}

	return errors.Join(errs...)
}

// ConfigureProgressManager sets the default values for the progress manager configuration
func ConfigureProgressManager(v *viper.Viper) {
	v.SetDefault("progressManager.expiration", "5m")
	v.SetDefault("progressManager.keyPrefix", "")
	redis.Configure(v, "progressManager.redis")
}
