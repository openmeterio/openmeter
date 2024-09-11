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
func ConfigureKafkaConfiguration(v *viper.Viper, prefix string) {
	// NOTE(chrisgacsal): make sure all the possible configuration parameters defaulted (even of the default is an empty string)
	// otherwise Viper might not register/resolve them.
	v.SetDefault(AddPrefix(prefix, "kafka.brokers"), "127.0.0.1:29092")
	v.SetDefault(AddPrefix(prefix, "kafka.securityProtocol"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.saslMechanisms"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.saslUsername"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.saslPassword"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.statsInterval"), 0)
	v.SetDefault(AddPrefix(prefix, "kafka.brokerAddressFamily"), "any")
	v.SetDefault(AddPrefix(prefix, "kafka.topicMetadataRefreshInterval"), 0)
	v.SetDefault(AddPrefix(prefix, "kafka.socketKeepAliveEnabled"), false)
	v.SetDefault(AddPrefix(prefix, "kafka.debugContexts"), nil)
	v.SetDefault(AddPrefix(prefix, "kafka.clientID"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.consumerGroupID"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.consumerGroupInstanceID"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.sessionTimeout"), 0)
	v.SetDefault(AddPrefix(prefix, "kafka.heartbeatInterval"), 0)
	v.SetDefault(AddPrefix(prefix, "kafka.enableAutoCommit"), true)
	v.SetDefault(AddPrefix(prefix, "kafka.enableAutoOffsetStore"), true)
	v.SetDefault(AddPrefix(prefix, "kafka.autoOffsetReset"), "")
	v.SetDefault(AddPrefix(prefix, "kafka.partitionAssignmentStrategy"), "cooperative-sticky")
}
