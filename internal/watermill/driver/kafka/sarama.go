package kafka

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/watermill/driver/kafka/metrics"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	defaultMeterPrefix = "sarama.publisher."
)

type BrokerConfiguration struct {
	KafkaConfig  config.KafkaConfiguration
	Logger       *slog.Logger
	MetricMeter  otelmetric.Meter
	MeterPrefix  string
	DebugLogging bool
	ClientID     string
}

func (c *BrokerConfiguration) Validate() error {
	if err := c.KafkaConfig.Validate(); err != nil {
		return fmt.Errorf("invalid kafka config: %w", err)
	}

	if c.ClientID == "" {
		return errors.New("client ID is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.MetricMeter == nil {
		return errors.New("metric meter is required")
	}

	return nil
}

func newSaramaConfig(in BrokerConfiguration) (*sarama.Config, error) {
	config := sarama.NewConfig()

	if in.MeterPrefix == "" {
		in.MeterPrefix = defaultMeterPrefix
	}

	config.Metadata.RefreshFrequency = in.KafkaConfig.TopicMetadataRefreshInterval.Duration()

	if in.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	config.ClientID = in.ClientID

	// These are globals, so we cannot append the publisher/subscriber name to them
	sarama.Logger = &SaramaLoggerAdaptor{
		loggerFunc: in.Logger.Info,
	}

	if in.DebugLogging {
		sarama.DebugLogger = &SaramaLoggerAdaptor{
			loggerFunc: in.Logger.Debug,
		}
	}

	if in.KafkaConfig.SecurityProtocol == "SASL_SSL" {
		config.Net.SASL.Enable = true
		config.Net.SASL.Handshake = true

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}

		switch in.KafkaConfig.SaslMechanisms {
		case "PLAIN":
			config.Net.SASL.User = in.KafkaConfig.SaslUsername
			config.Net.SASL.Password = in.KafkaConfig.SaslPassword
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		default:
			return nil, fmt.Errorf("unsupported SASL mechanism: %s", in.KafkaConfig.SaslMechanisms)
		}
	}

	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true

	meterRegistry, err := metrics.NewRegistry(metrics.NewRegistryOptions{
		MetricMeter:     in.MetricMeter,
		NameTransformFn: metrics.MetricAddNamePrefix(in.MeterPrefix),
		ErrorHandler:    metrics.LoggingErrorHandler(in.Logger),
	})
	if err != nil {
		return nil, err
	}

	config.MetricRegistry = meterRegistry
	return config, nil
}
