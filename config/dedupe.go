package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/internal/dedupe/redisdedupe"
	"github.com/openmeterio/openmeter/pkg/redis"
)

// Requires [mapstructurex.MapDecoderHookFunc] to be high up in the decode hook chain.
type DedupeConfiguration struct {
	Enabled bool

	DedupeDriverConfiguration
}

func (c DedupeConfiguration) NewDeduplicator() (dedupe.Deduplicator, error) {
	if !c.Enabled {
		return nil, errors.New("dedupe: disabled")
	}

	if c.DedupeDriverConfiguration == nil {
		return nil, errors.New("dedupe: missing driver configuration")
	}

	return c.DedupeDriverConfiguration.NewDeduplicator()
}

func (c DedupeConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.DedupeDriverConfiguration == nil {
		return errors.New("missing driver configuration")
	}

	if err := c.DedupeDriverConfiguration.Validate(); err != nil {
		return fmt.Errorf("driver(%s): %w", c.DedupeDriverConfiguration.DriverName(), err)
	}

	return nil
}

type rawDedupeConfiguration struct {
	Enabled bool
	Driver  string
	Config  map[string]any
}

func (c *DedupeConfiguration) DecodeMap(v map[string]any) error {
	var rawConfig rawDedupeConfiguration

	err := mapstructure.Decode(v, &rawConfig)
	if err != nil {
		return err
	}

	c.Enabled = rawConfig.Enabled

	// Deduplication is disabled and not configured, so skip further decoding
	if !c.Enabled && rawConfig.Driver == "" {
		return nil
	}

	switch rawConfig.Driver {
	case "memory":
		var driverConfig DedupeDriverMemoryConfiguration

		err := mapstructure.Decode(rawConfig.Config, &driverConfig)
		if err != nil {
			return fmt.Errorf("dedupe: decoding memory driver config: %w", err)
		}

		c.DedupeDriverConfiguration = driverConfig

	case "redis":
		var driverConfig DedupeDriverRedisConfiguration

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           &driverConfig,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
			),
		})
		if err != nil {
			return fmt.Errorf("dedupe: creating decoder: %w", err)
		}

		err = decoder.Decode(rawConfig.Config)
		if err != nil {
			return fmt.Errorf("dedupe: decoding redis driver config: %w", err)
		}

		c.DedupeDriverConfiguration = driverConfig

	default:
		c.DedupeDriverConfiguration = unknownDedupeDriverConfiguration{
			name: rawConfig.Driver,
		}
	}

	return nil
}

type DedupeDriverConfiguration interface {
	DriverName() string
	NewDeduplicator() (dedupe.Deduplicator, error)
	Validate() error
}

type unknownDedupeDriverConfiguration struct {
	name string
}

func (c unknownDedupeDriverConfiguration) DriverName() string {
	return c.name
}

func (c unknownDedupeDriverConfiguration) NewDeduplicator() (dedupe.Deduplicator, error) {
	return nil, errors.New("dedupe: unknown driver")
}

func (c unknownDedupeDriverConfiguration) Validate() error {
	return errors.New("unknown driver")
}

// Dedupe memory driver configuration
type DedupeDriverMemoryConfiguration struct {
	Enabled bool
	Size    int
}

func (DedupeDriverMemoryConfiguration) DriverName() string {
	return "memory"
}

func (c DedupeDriverMemoryConfiguration) NewDeduplicator() (dedupe.Deduplicator, error) {
	return memorydedupe.NewDeduplicator(c.Size)
}

func (c DedupeDriverMemoryConfiguration) Validate() error {
	if c.Size == 0 {
		return errors.New("size is required")
	}

	return nil
}

// Dedupe redis driver configuration
type DedupeDriverRedisConfiguration struct {
	redis.Config `mapstructure:",squash"`

	Expiration time.Duration
}

func (DedupeDriverRedisConfiguration) DriverName() string {
	return "redis"
}

func (c DedupeDriverRedisConfiguration) NewDeduplicator() (dedupe.Deduplicator, error) {
	redisClient, err := redis.NewClient(redis.Options{Config: c.Config})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis client: %w", err)
	}

	// TODO: register health check for redis
	return redisdedupe.Deduplicator{
		Redis:      redisClient,
		Expiration: c.Expiration,
	}, nil
}

func (c DedupeDriverRedisConfiguration) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	if c.Sentinel.Enabled {
		if c.Sentinel.MasterName == "" {
			return errors.New("sentinel: master name is required")
		}
	}

	return nil
}

// ConfigureDedupe configures some defaults in the Viper instance.
func ConfigureDedupe(v *viper.Viper) {
	v.SetDefault("dedupe.enabled", false)
	v.SetDefault("dedupe.driver", "memory")

	// Memory driver
	v.SetDefault("dedupe.config.size", 128)

	// Redis driver
	v.SetDefault("dedupe.config.address", "127.0.0.1:6379")
	v.SetDefault("dedupe.config.database", 0)
	v.SetDefault("dedupe.config.username", "")
	v.SetDefault("dedupe.config.password", "")
	v.SetDefault("dedupe.config.expiration", "24h")
	v.SetDefault("dedupe.config.sentinel.enabled", false)
	v.SetDefault("dedupe.config.sentinel.masterName", "")
	v.SetDefault("dedupe.config.tls.enabled", false)
	v.SetDefault("dedupe.config.tls.insecureSkipVerify", false)
}
