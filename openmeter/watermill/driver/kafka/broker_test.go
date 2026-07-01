package kafka

import (
	"log/slog"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/config"
)

func TestBrokerOptions_createKafkaConfig_SASL(t *testing.T) {
	tests := []struct {
		name                 string
		kafkaConfig          config.KafkaConfiguration
		expectSASLEnabled    bool
		expectTLSEnabled     bool
		expectSCRAMGenerator bool
	}{
		{
			name: "PLAINTEXT does not enable SASL or TLS",
			kafkaConfig: config.KafkaConfiguration{
				Broker: "localhost:9092",
			},
			expectSASLEnabled: false,
			expectTLSEnabled:  false,
		},
		{
			name: "SASL_PLAINTEXT enables SASL without TLS",
			kafkaConfig: config.KafkaConfiguration{
				Broker:           "localhost:9092",
				SecurityProtocol: "SASL_PLAINTEXT",
				SaslMechanisms:   sarama.SASLTypePlaintext,
				SaslUsername:     "user",
				SaslPassword:     "pass",
			},
			expectSASLEnabled: true,
			expectTLSEnabled:  false,
		},
		{
			name: "SASL_SSL enables both SASL and TLS",
			kafkaConfig: config.KafkaConfiguration{
				Broker:           "localhost:9092",
				SecurityProtocol: "SASL_SSL",
				SaslMechanisms:   sarama.SASLTypePlaintext,
				SaslUsername:     "user",
				SaslPassword:     "pass",
			},
			expectSASLEnabled: true,
			expectTLSEnabled:  true,
		},
		{
			name: "SASL_PLAINTEXT with SCRAM wires up a SCRAM client generator",
			kafkaConfig: config.KafkaConfiguration{
				Broker:           "localhost:9092",
				SecurityProtocol: "SASL_PLAINTEXT",
				SaslMechanisms:   sarama.SASLTypeSCRAMSHA512,
				SaslUsername:     "user",
				SaslPassword:     "pass",
			},
			expectSASLEnabled:    true,
			expectTLSEnabled:     false,
			expectSCRAMGenerator: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &BrokerOptions{
				KafkaConfig: tt.kafkaConfig,
				ClientID:    "test-client",
				Logger:      slog.Default(),
				MetricMeter: noop.NewMeterProvider().Meter("test"),
			}

			cfg, err := o.createKafkaConfig("publisher")
			require.NoError(t, err)

			assert.Equal(t, tt.expectSASLEnabled, cfg.Net.SASL.Enable)
			assert.Equal(t, tt.expectTLSEnabled, cfg.Net.TLS.Enable)

			if tt.expectSASLEnabled {
				assert.Equal(t, tt.kafkaConfig.SaslUsername, cfg.Net.SASL.User)
				assert.Equal(t, tt.kafkaConfig.SaslPassword, cfg.Net.SASL.Password)
				assert.Equal(t, sarama.SASLMechanism(tt.kafkaConfig.SaslMechanisms), cfg.Net.SASL.Mechanism)
			}

			if tt.expectSCRAMGenerator {
				assert.NotNil(t, cfg.Net.SASL.SCRAMClientGeneratorFunc)
			} else {
				assert.Nil(t, cfg.Net.SASL.SCRAMClientGeneratorFunc)
			}
		})
	}
}
