package config

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/redis"
)

type BalanceWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
	StateStorage          BalanceWorkerStateStorageConfiguration
}

func (c BalanceWorkerConfiguration) Validate() error {
	var errs []error

	if err := c.ConsumerConfiguration.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "consumer"))
	}

	if err := c.StateStorage.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "stateStorage"))
	}

	return errors.Join(errs...)
}

type BalanceWorkerStateStorageDriver string

const (
	BalanceWorkerStateStorageDriverRedis    BalanceWorkerStateStorageDriver = "redis"
	BalanceWorkerStateStorageDriverInMemory BalanceWorkerStateStorageDriver = "in-memory"
)

type rawBalanceWorkerStateStorageConfiguration struct {
	Driver BalanceWorkerStateStorageDriver
	Config map[string]any
}

type BalanceWorkerStateStorageConfiguration struct {
	Driver BalanceWorkerStateStorageDriver

	BalanceWorkerStateStorageBackendConfiguration
}

func (c *BalanceWorkerStateStorageConfiguration) DecodeMap(v map[string]any) error {
	var raw rawBalanceWorkerStateStorageConfiguration

	if err := mapstructure.Decode(v, &raw); err != nil {
		return err
	}

	c.Driver = raw.Driver

	switch c.Driver {
	case BalanceWorkerStateStorageDriverRedis:
		var redisConfig BalanceWorkerStateStorageRedisBackendConfiguration

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           &redisConfig,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
			),
		})
		if err != nil {
			return err
		}

		if err := decoder.Decode(raw.Config); err != nil {
			return err
		}

		c.BalanceWorkerStateStorageBackendConfiguration = redisConfig
	case BalanceWorkerStateStorageDriverInMemory:
		// no config
	}

	return nil
}

func (c BalanceWorkerStateStorageConfiguration) Validate() error {
	errs := []error{}
	if !slices.Contains([]BalanceWorkerStateStorageDriver{
		BalanceWorkerStateStorageDriverRedis,
		BalanceWorkerStateStorageDriverInMemory,
	}, c.Driver) {
		errs = append(errs, fmt.Errorf("invalid driver: %s", c.Driver))
	}

	if c.Driver == BalanceWorkerStateStorageDriverRedis {
		if c.BalanceWorkerStateStorageBackendConfiguration == nil {
			errs = append(errs, errors.New("state storage backend configuration is required"))
		} else {
			if err := c.BalanceWorkerStateStorageBackendConfiguration.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("state storage backend: %w", err))
			}
		}
	}

	return errors.Join(errs...)
}

func (c BalanceWorkerStateStorageConfiguration) GetRedisBackendConfiguration() (BalanceWorkerStateStorageRedisBackendConfiguration, error) {
	if c.Driver != BalanceWorkerStateStorageDriverRedis {
		return BalanceWorkerStateStorageRedisBackendConfiguration{}, fmt.Errorf("driver is not redis")
	}

	if c.BalanceWorkerStateStorageBackendConfiguration == nil {
		return BalanceWorkerStateStorageRedisBackendConfiguration{}, errors.New("state storage backend configuration is required")
	}

	redisConfig, ok := c.BalanceWorkerStateStorageBackendConfiguration.(BalanceWorkerStateStorageRedisBackendConfiguration)
	if !ok {
		return BalanceWorkerStateStorageRedisBackendConfiguration{}, fmt.Errorf("state storage backend configuration is not a redis configuration")
	}

	return redisConfig, nil
}

type BalanceWorkerStateStorageBackendConfiguration interface {
	models.Validator
}

type BalanceWorkerStateStorageRedisBackendConfiguration struct {
	redis.Config `mapstructure:",squash"`
	Expiration   time.Duration
}

func (c BalanceWorkerStateStorageRedisBackendConfiguration) Validate() error {
	errs := []error{}

	if c.Expiration <= 0 {
		errs = append(errs, errors.New("expiration should be greater than 0"))
	}

	if err := c.Config.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("redis: %w", err))
	}

	return errors.Join(errs...)
}

func ConfigureBalanceWorker(v *viper.Viper) {
	ConfigureConsumer(v, "balanceWorker")
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")

	v.SetDefault("balanceWorker.stateStorage.driver", BalanceWorkerStateStorageDriverInMemory)
	v.SetDefault("balanceWorker.stateStorage.config.expiration", 24*time.Hour)
}
