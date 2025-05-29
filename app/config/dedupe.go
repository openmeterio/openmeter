package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/openmeter/dedupe/redisdedupe"
	"github.com/openmeterio/openmeter/pkg/errorsx"
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
		return errorsx.WithPrefix(err, fmt.Sprintf("driver(%s)", c.DriverName()))
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
	var errs []error

	if c.Size == 0 {
		errs = append(errs, errors.New("size is required"))
	}

	return errors.Join(errs...)
}

// Dedupe redis driver configuration
type DedupeDriverRedisConfiguration struct {
	redis.Config `mapstructure:",squash"`

	Expiration time.Duration
	Mode       redisdedupe.DedupeMode
}

func (DedupeDriverRedisConfiguration) DriverName() string {
	return "redis"
}

func (c DedupeDriverRedisConfiguration) NewDeduplicator() (dedupe.Deduplicator, error) {
	redisClient, err := c.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis client: %w", err)
	}

	// TODO: register health check for redis
	return redisdedupe.Deduplicator{
		Redis:      redisClient,
		Expiration: c.Expiration,
		Mode:       c.Mode,
	}, nil
}

func (c DedupeDriverRedisConfiguration) Validate() error {
	var errs []error

	if err := c.Config.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "redis"))
	}

	if err := c.Mode.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "mode"))
	}

	return errors.Join(errs...)
}

// ConfigureDedupe configures some defaults in the Viper instance.
func ConfigureDedupe(v *viper.Viper) {
	v.SetDefault("dedupe.enabled", false)
	v.SetDefault("dedupe.driver", "memory")

	// Memory driver
	v.SetDefault("dedupe.config.size", 128)

	// Redis driver
	redis.Configure(v, "dedupe.config")
	v.SetDefault("sink.dedupe.config.mode", redisdedupe.DedupeModeRawKey)
	v.SetDefault("dedupe.config.mode", redisdedupe.DedupeModeRawKey)
}
