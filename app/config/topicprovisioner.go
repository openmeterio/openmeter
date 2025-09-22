package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// TopicProvisionerConfig stores the configuration for TopicProvisioner
type TopicProvisionerConfig struct {
	// The maximum number of entries stored in topic cache at a time which after the least recently used is evicted.
	// Setting size to 0 makes it unlimited
	CacheSize int

	// The maximum time an entries is kept in cache before being evicted
	CacheTTL time.Duration

	// ProtectedTopics defines a list of topics which are protected from deletion.
	ProtectedTopics []string

	// Enabled defines whether topic provisioning is enabled or not.
	Enabled bool
}

func (c TopicProvisionerConfig) Validate() error {
	var errs []error

	if !c.Enabled {
		return nil
	}

	if c.CacheSize < 0 {
		errs = append(errs, fmt.Errorf("invalid cache size: %d", c.CacheSize))
	}

	if c.CacheTTL < 0 {
		errs = append(errs, fmt.Errorf("invalid cache ttl: %d", c.CacheTTL))
	}

	return errors.Join(errs...)
}

// ConfigureTopicProvisioner configures some defaults in the Viper instance.
func ConfigureTopicProvisioner(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("cacheSize"), 250)
	v.SetDefault(prefixer("cacheTTL"), "5m")
	v.SetDefault(prefixer("protectedTopics"), nil)
	v.SetDefault(prefixer("enabled"), true)
}
