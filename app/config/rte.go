package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type RTEConfiguration struct {
	ConsumerConfiguration `mapstructure:",squash"`
	PollInterval          time.Duration
	CalculationTimeout    time.Duration
	// Enable on billing and ledger writers when RTE tracks changes to credit balances.
	PublishSourceEvents bool
}

func (c RTEConfiguration) Validate() error {
	var errs []error
	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be positive"))
	}

	if c.CalculationTimeout <= 0 {
		errs = append(errs, errors.New("calculation timeout must be positive"))
	}

	return errors.Join(c.ConsumerConfiguration.Validate(), errors.Join(errs...))
}

func ConfigureRTE(v *viper.Viper) {
	v.SetDefault("rte.pollInterval", time.Second)
	v.SetDefault("rte.calculationTimeout", 30*time.Second)
	v.SetDefault("rte.publishSourceEvents", false)
	v.SetDefault("rte.dlq.topic", "om_sys.rte_dlq")
	v.SetDefault("rte.consumerGroupName", "om_rte")

	ConfigureConsumer(v, "rte")
}
