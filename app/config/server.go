package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

// ServerConfig holds HTTP server timeout configuration.
type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func (c ServerConfig) Validate() error {
	var errs []error

	if c.ReadHeaderTimeout < 0 {
		errs = append(errs, errors.New("readHeaderTimeout must be non-negative"))
	}

	if c.ReadTimeout < 0 {
		errs = append(errs, errors.New("readTimeout must be non-negative"))
	}

	if c.WriteTimeout < 0 {
		errs = append(errs, errors.New("writeTimeout must be non-negative"))
	}

	if c.IdleTimeout < 0 {
		errs = append(errs, errors.New("idleTimeout must be non-negative"))
	}

	return errors.Join(errs...)
}

// ConfigureServer sets defaults for HTTP server timeouts.
func ConfigureServer(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("readHeaderTimeout"), 10*time.Second)
	v.SetDefault(prefixer("readTimeout"), 60*time.Second)
	v.SetDefault(prefixer("writeTimeout"), 90*time.Second)
	v.SetDefault(prefixer("idleTimeout"), 120*time.Second)
}
