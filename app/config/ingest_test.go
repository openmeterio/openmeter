package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			Name: "SslCaLocation",
			KafkaConfig: KafkaConfiguration{
				Broker:           "127.0.0.1:29092",
				SecurityProtocol: "SSL",
				SslCaLocation:    "/etc/openmeter/kafka-ca.pem",
			},
			ExpectedKafkaConfigMap: kafka.ConfigMap{
				"bootstrap.servers":      "127.0.0.1:29092",
				"broker.address.family":  pkgkafka.BrokerAddressFamilyIPv4,
				"go.logs.channel.enable": true,
				"security.protocol":      "SSL",
				"ssl.ca.location":        "/etc/openmeter/kafka-ca.pem",
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

func TestKafkaConfigurationSslCaLocationValidate(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caFile, []byte("x"), 0o600))

	require.NoError(t, KafkaConfiguration{Broker: "b:9092"}.Validate())
	require.NoError(t, KafkaConfiguration{Broker: "b:9092", SslCaLocation: caFile}.Validate())
	require.ErrorContains(t, KafkaConfiguration{Broker: "b:9092", SslCaLocation: caFile + ".missing"}.Validate(), "invalid ssl ca location")
}

// TestKafkaIngestSslCaLocationEnv verifies the viper key and the environment variable that feed SslCaLocation.
func TestKafkaIngestSslCaLocationEnv(t *testing.T) {
	t.Setenv("INGEST_KAFKA_SSLCALOCATION", "/etc/ssl/kafka-ca.pem")
	t.Setenv("SINK_KAFKA_SSLCALOCATION", "/etc/ssl/sink-ca.pem")

	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ContinueOnError)
	SetViperDefaults(v, flags)

	var cfg Configuration

	require.NoError(t, v.Unmarshal(&cfg, viper.DecodeHook(DecodeHook())))
	assert.Equal(t, "/etc/ssl/kafka-ca.pem", cfg.Ingest.Kafka.SslCaLocation)
	assert.Equal(t, "/etc/ssl/sink-ca.pem", cfg.Sink.Kafka.SslCaLocation)
}
