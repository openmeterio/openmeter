package config

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"

	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

func TestKafkaIngestConfiguration(t *testing.T) {
	tests := []struct {
		Name string

		KafkaConfig            KafkaConfiguration
		ExpectedKafkaConfigMap kafka.ConfigMap
	}{
		{
			Name: "All",
			KafkaConfig: KafkaConfiguration{
				Broker:                       "127.0.0.1:29092",
				SecurityProtocol:             "SASL_SSL",
				SaslMechanisms:               "PLAIN",
				SaslUsername:                 "user",
				SaslPassword:                 "pass",
				StatsInterval:                pkgkafka.TimeDurationMilliSeconds(10 * time.Second),
				BrokerAddressFamily:          "v6",
				SocketKeepAliveEnabled:       true,
				TopicMetadataRefreshInterval: pkgkafka.TimeDurationMilliSeconds(time.Minute),
				DebugContexts: pkgkafka.DebugContexts{
					"broker",
					"topic",
					"consumer",
				},
			},
			ExpectedKafkaConfigMap: kafka.ConfigMap{
				"bootstrap.servers":                  "127.0.0.1:29092",
				"broker.address.family":              pkgkafka.BrokerAddressFamilyIPv6,
				"go.logs.channel.enable":             true,
				"metadata.max.age.ms":                pkgkafka.TimeDurationMilliSeconds(3 * time.Minute),
				"sasl.mechanism":                     "PLAIN",
				"sasl.password":                      "pass",
				"sasl.username":                      "user",
				"security.protocol":                  "SASL_SSL",
				"socket.keepalive.enable":            true,
				"statistics.interval.ms":             pkgkafka.TimeDurationMilliSeconds(10 * time.Second),
				"topic.metadata.refresh.interval.ms": pkgkafka.TimeDurationMilliSeconds(time.Minute),
				"debug":                              "broker,topic,consumer",
			},
		},
		{
			Name: "Basic",
			KafkaConfig: KafkaConfiguration{
				Broker: "127.0.0.1:29092",
			},
			ExpectedKafkaConfigMap: kafka.ConfigMap{
				"bootstrap.servers":      "127.0.0.1:29092",
				"broker.address.family":  pkgkafka.BrokerAddressFamilyIPv4,
				"go.logs.channel.enable": true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			config := test.KafkaConfig.CreateKafkaConfig()
			assert.Equal(t, test.ExpectedKafkaConfigMap, config)
		})
	}
}

func TestKafkaConfigurationValidate(t *testing.T) {
	tests := []struct {
		Name        string
		KafkaConfig KafkaConfiguration
		ExpectError bool
	}{
		{
			Name: "Plaintext without SASL is valid",
			KafkaConfig: KafkaConfiguration{
				Broker: "127.0.0.1:29092",
			},
			ExpectError: false,
		},
		{
			Name: "SASL_PLAINTEXT with full credentials is valid",
			KafkaConfig: KafkaConfiguration{
				Broker:           "127.0.0.1:29092",
				SecurityProtocol: "SASL_PLAINTEXT",
				SaslMechanisms:   "PLAIN",
				SaslUsername:     "user",
				SaslPassword:     "pass",
			},
			ExpectError: false,
		},
		{
			Name: "SASL_SSL with empty mechanism but full credentials is valid",
			KafkaConfig: KafkaConfiguration{
				Broker:           "127.0.0.1:29092",
				SecurityProtocol: "SASL_SSL",
				SaslUsername:     "user",
				SaslPassword:     "pass",
			},
			ExpectError: false,
		},
		{
			Name: "SASL_PLAINTEXT without credentials is invalid",
			KafkaConfig: KafkaConfiguration{
				Broker:           "127.0.0.1:29092",
				SecurityProtocol: "SASL_PLAINTEXT",
			},
			ExpectError: true,
		},
		{
			Name: "SASL_SSL missing password is invalid",
			KafkaConfig: KafkaConfiguration{
				Broker:           "127.0.0.1:29092",
				SecurityProtocol: "SASL_SSL",
				SaslMechanisms:   "PLAIN",
				SaslUsername:     "user",
			},
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := test.KafkaConfig.Validate()
			if test.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
