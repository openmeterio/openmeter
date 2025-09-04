package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// SubjectConfig represents the configuration for the subject.
type SubjectConfig struct {
	Manager SubjectManagerConfig `mapstructure:"manager"`
}

// Validate validates the configuration.
func (c SubjectConfig) Validate() error {
	var errs []error

	if err := c.Manager.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("manager: %w", err))
	}

	return errors.Join(errs...)
}

// SubjectManagerConfig represents the configuration for the subject manager.
type SubjectManagerConfig struct {
	CacheReloadInterval time.Duration `mapstructure:"cacheReloadInterval"`
	CacheReloadTimeout  time.Duration `mapstructure:"cacheReloadTimeout"`
	CachePrefillCount   int           `mapstructure:"cachePrefillCount"`
	CacheSize           int           `mapstructure:"cacheSize"`
	PaginationSize      int           `mapstructure:"paginationSize"`
}

// Validate validates the configuration.
func (c SubjectManagerConfig) Validate() error {
	var errs []error

	if c.CacheReloadInterval == 0 {
		errs = append(errs, errors.New("cache reload interval is required"))
	}

	if c.CacheReloadTimeout <= 0 {
		errs = append(errs, errors.New("cache reload timeout must be greater than 0"))
	}

	if c.CachePrefillCount <= 0 {
		errs = append(errs, errors.New("cache prefill count must be greater than 0"))
	}

	if c.CachePrefillCount > c.CacheSize {
		errs = append(errs, errors.New("cache prefill count must be less than or equal to cache size"))
	}

	if c.CacheSize <= 0 {
		errs = append(errs, errors.New("cache size must be greater than 0"))
	}

	if c.PaginationSize <= 0 {
		errs = append(errs, errors.New("pagination size must be greater than 0"))
	}

	return errors.Join(errs...)
}

// SetViperDefaults sets the default values for the configuration.
func ConfigureSubject(v *viper.Viper) {
	v.SetDefault("subject.manager.cacheReloadInterval", "5m")
	v.SetDefault("subject.manager.cacheReloadTimeout", "2m")
	v.SetDefault("subject.manager.cachePrefillCount", 250_000)
	v.SetDefault("subject.manager.cacheSize", 1_000_000)
	v.SetDefault("subject.manager.paginationSize", 10000)
}
