package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type RTEConfiguration struct {
	ConsumerConfiguration  `mapstructure:",squash"`
	PollInterval           time.Duration
	CalculationTimeout     time.Duration
	CalculationConcurrency int
	Enabled                bool
	ShadowSamplePercent    int
}

func (c RTEConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	var errs []error
	if c.ShadowSamplePercent < 1 || c.ShadowSamplePercent > 100 {
		errs = append(errs, errors.New("shadow sample percent must be between 1 and 100"))
	}

	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be positive"))
	}

	if c.CalculationConcurrency <= 0 {
		errs = append(errs, errors.New("calculation concurrency must be positive"))
	}

	if c.CalculationTimeout <= 0 {
		errs = append(errs, errors.New("calculation timeout must be positive"))
	}

	return errors.Join(c.ConsumerConfiguration.Validate(), errors.Join(errs...))
}

func ConfigureRTE(v *viper.Viper) {
	v.SetDefault("rte.pollInterval", time.Second)
	v.SetDefault("rte.calculationTimeout", 30*time.Second)
	v.SetDefault("rte.calculationConcurrency", 4)
	v.SetDefault("rte.enabled", false)
	v.SetDefault("rte.shadowSamplePercent", 10)
	v.SetDefault("rte.dlq.topic", "om_sys.rte_dlq")
	v.SetDefault("rte.consumerGroupName", "om_rte")

	ConfigureConsumer(v, "rte")
}
