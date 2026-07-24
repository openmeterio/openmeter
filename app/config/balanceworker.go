package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type BalanceWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
	StateStorage          BalanceWorkerStateStorageConfiguration
	UseWatermill          bool
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
	HighWatermarkCache BalanceWorkerHighWatermarkCacheConfiguration
}

func (c BalanceWorkerStateStorageConfiguration) Validate() error {
	var errs []error

	if err := c.HighWatermarkCache.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "high watermark cache"))
	}

	return errors.Join(errs...)
}

type BalanceWorkerHighWatermarkCacheConfiguration struct {
	LRUCacheSize int
}

func (c BalanceWorkerHighWatermarkCacheConfiguration) Validate() error {
	if c.LRUCacheSize <= 0 {
		return errors.New("LRU cache size must be positive")
	}

	return nil
}

func ConfigureBalanceWorker(v *viper.Viper) {
	ConfigureConsumer(v, "balanceWorker")
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")

	v.SetDefault("balanceWorker.stateStorage.highWatermarkCache.lruCacheSize", 100_000)
	v.SetDefault("balanceWorker.useWatermill", true)
}
