package config

import (
	"github.com/spf13/viper"
)

type BalanceWorkerConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
}

func (c BalanceWorkerConfiguration) Validate() error {
	if err := c.ConsumerConfiguration.Validate(); err != nil {
		return err
	}

	return nil
}

func ConfigureBalanceWorker(v *viper.Viper) {
	ConfigureConsumer(v, "balanceWorker")
	v.SetDefault("balanceWorker.dlq.topic", "om_sys.balance_worker_dlq")
	v.SetDefault("balanceWorker.consumerGroupName", "om_balance_worker")
}
