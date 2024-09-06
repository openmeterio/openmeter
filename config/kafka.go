package config

import (
	"github.com/spf13/viper"

	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var _ pkgkafka.ConfigValidator = (*KafkaConfig)(nil)

type KafkaConfig struct {
	pkgkafka.CommonConfigParams   `mapstructure:",squash"`
	pkgkafka.ConsumerConfigParams `mapstructure:",squash"`
	pkgkafka.ProducerConfigParams `mapstructure:",squash"`
}

func (c KafkaConfig) AsProducerConfig() pkgkafka.ProducerConfig {
	return pkgkafka.ProducerConfig{
		CommonConfigParams:   c.CommonConfigParams,
		ProducerConfigParams: c.ProducerConfigParams,
	}
}

func (c KafkaConfig) AsConsumerConfig() pkgkafka.ConsumerConfig {
	return pkgkafka.ConsumerConfig{
		CommonConfigParams:   c.CommonConfigParams,
		ConsumerConfigParams: c.ConsumerConfigParams,
	}
}

func (c KafkaConfig) Validate() error {
	validators := []pkgkafka.ConfigValidator{
		c.CommonConfigParams,
		c.ConsumerConfigParams,
		c.ProducerConfigParams,
	}

	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ConfigureKafkaConfiguration sets defaults in the Viper instance.
func ConfigureKafkaConfiguration(v *viper.Viper) {
	v.SetDefault("kafka.brokers", "127.0.0.1:29092")
}
