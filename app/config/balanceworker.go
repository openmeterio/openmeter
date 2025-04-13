package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type BalanceWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`

	Estimator EstimatorConfiguration
}

func (c BalanceWorkerConfiguration) Validate() error {
	var errs []error

	if err := c.ConsumerConfiguration.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "consumer"))
	}

	return errors.Join(errs...)
}

type EstimatorConfiguration struct {
	Enabled        bool
	RedisURL       string
	ValidationRate float64
	LockTimeout    time.Duration
}

func (c EstimatorConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error

	if c.RedisURL == "" {
		errs = append(errs, errors.New("redis url is required"))
	}

	if c.ValidationRate <= 0 || c.ValidationRate > 1 {
		errs = append(errs, errors.New("validation rate must be between 0 and 1"))
	}

	if c.LockTimeout <= 0 {
		errs = append(errs, errors.New("lock timeout must be greater than 0"))
	}

	return errors.Join(errs...)
}

func ConfigureBalanceWorker(v *viper.Viper) {
	ConfigureConsumer(v, "balanceWorker")
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")

	v.SetDefault("balanceWorker.estimator.enabled", false)
	v.SetDefault("balanceWorker.estimator.redisURL", "redis://localhost:6379")
	v.SetDefault("balanceWorker.estimator.validationRate", 0.01) // 1%
	v.SetDefault("balanceWorker.estimator.lockTimeout", 3*time.Second)
}
