package kafka

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
)

const (
	defaultKeepalive = time.Minute
)

type BrokerOptions struct {
	KafkaConfig config.KafkaConfiguration
	ClientID    string
	Logger      *slog.Logger
	MetricMeter otelmetric.Meter
}

func (o *BrokerOptions) Validate() error {
	if err := o.KafkaConfig.Validate(); err != nil {
		return fmt.Errorf("invalid kafka config: %w", err)
	}

	if o.ClientID == "" {
		return errors.New("client ID is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	if o.MetricMeter == nil {
		return errors.New("metric meter is required")
	}
	return nil
}

func (o *BrokerOptions) createKafkaConfig(role string) (*sarama.Config, error) {
	config := sarama.NewConfig()

	if role == "" {
		return nil, errors.New("role is required")
	}
	if o.KafkaConfig.SocketKeepAliveEnabled {
		config.Net.KeepAlive = defaultKeepalive
	}
	config.Metadata.RefreshFrequency = o.KafkaConfig.TopicMetadataRefreshInterval.Duration()
	if o.ClientID == "" {
		return nil, errors.New("client ID is required")
	}
	config.ClientID = fmt.Sprintf("%s-%s", o.ClientID, role)

	// These are globals, so we cannot append the publisher/subscriber name to them
	logger := o.Logger.With(slog.String(string(semconv.OTelScopeNameKey), "sarama"))

	sarama.Logger = &SaramaLoggerAdaptor{
		loggerFunc: logger.Info,
	}

	// NOTE: always set the sarama.DebugLogger otherwise the debug level logs are redirected to the sarama.Logger by default
	sarama.DebugLogger = &SaramaLoggerAdaptor{
		loggerFunc: logger.Debug,
	}

	if o.KafkaConfig.SecurityProtocol == "SASL_SSL" {
		config.Net.SASL.Enable = true
		config.Net.SASL.Handshake = true

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}

		switch o.KafkaConfig.SaslMechanisms {
		case sarama.SASLTypePlaintext:
			config.Net.SASL.User = o.KafkaConfig.SaslUsername
			config.Net.SASL.Password = o.KafkaConfig.SaslPassword
			config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)
		case sarama.SASLTypeSCRAMSHA256:
			config.Net.SASL.User = o.KafkaConfig.SaslUsername
			config.Net.SASL.Password = o.KafkaConfig.SaslPassword
			config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			}
		case sarama.SASLTypeSCRAMSHA512:
			config.Net.SASL.User = o.KafkaConfig.SaslUsername
			config.Net.SASL.Password = o.KafkaConfig.SaslPassword
			config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			}
		default:
			return nil, fmt.Errorf("unsupported SASL mechanism: %s", o.KafkaConfig.SaslMechanisms)
		}
	}

	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true

	meterRegistry, err := metrics.NewRegistry(metrics.NewRegistryOptions{
		MetricMeter:     o.MetricMeter,
		NameTransformFn: SaramaMetricRenamer(role),
		ErrorHandler:    metrics.LoggingErrorHandler(o.Logger),
	})
	if err != nil {
		return nil, err
	}

	config.MetricRegistry = meterRegistry

	return config, nil
}
