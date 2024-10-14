package config

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type BalanceWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
}

func (c BalanceWorkerConfiguration) Validate() error {
	var errs []error

	if err := c.ConsumerConfiguration.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "consumer"))
	}

	return errors.Join(errs...)
}

func ConfigureBalanceWorker(v *viper.Viper) {
	ConfigureConsumer(v, "balanceWorker")
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")
}
