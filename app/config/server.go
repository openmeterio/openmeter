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

	ResponseValidation ResponseValidationConfig
}

// ResponseValidationConfig controls optional post-response OpenAPI validation on the v3 API.
type ResponseValidationConfig struct {
	Mode ResponseValidationMode
}

type ResponseValidationMode string

const (
	// ResponseValidationModeOff disables response validation. This is the default.
	ResponseValidationModeOff ResponseValidationMode = "off"
	// ResponseValidationModeUnstable validates only routes marked x-unstable: true in the spec.
	ResponseValidationModeUnstable ResponseValidationMode = "unstable"
	// ResponseValidationModeAll validates every route in the v3 spec.
	ResponseValidationModeAll ResponseValidationMode = "all"
)

func (m ResponseValidationMode) Enabled() bool {
	return m != "" && m != ResponseValidationModeOff
}

func (m ResponseValidationMode) Validate() error {
	switch m {
	case "", ResponseValidationModeOff, ResponseValidationModeUnstable, ResponseValidationModeAll:
		return nil
	default:
		return errors.New("invalid response validation mode (allowed: off, unstable, all)")
	}
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

	if err := c.ResponseValidation.Mode.Validate(); err != nil {
		errs = append(errs, err)
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

	v.SetDefault(prefixer("responseValidation.mode"), string(ResponseValidationModeOff))
}
