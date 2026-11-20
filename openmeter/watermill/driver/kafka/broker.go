package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

// applyCACertificates sets the trusted root CAs of tlsConfig from KafkaConfig.SslCaLocation, if configured.
//
// Like librdkafka's ssl.ca.location (used by the ingest producer/consumer), the configured file replaces the
// default trust store instead of extending it, so both Kafka clients trust the same CAs. When no location is
// configured tlsConfig is left untouched and the system trust store is used.
func (o *BrokerOptions) applyCACertificates(tlsConfig *tls.Config) error {
	if o.KafkaConfig.SslCaLocation == "" {
		return nil
	}

	pemData, err := os.ReadFile(o.KafkaConfig.SslCaLocation)
	if err != nil {
		return fmt.Errorf("failed to read kafka ssl ca location: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return fmt.Errorf("kafka ssl ca location %q does not contain any valid PEM certificates", o.KafkaConfig.SslCaLocation)
	}

	tlsConfig.RootCAs = pool

	return nil
}

func (o *BrokerOptions) createKafkaConfig(role string) (*sarama.Config, error) {
	config := sarama.NewConfig()

	// ApiVersions negotiation is disabled below, so this is the request version
	// the broker must accept. 2.6.2 is the newest protocol a Kafka 2.6.2 broker
	// supports, and it remains valid for 3.x and 4.x brokers.
	config.Version = sarama.V2_6_2_0

	if role == "" {
		return nil, errors.New("role is required")
	}

	if o.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	if o.KafkaConfig.SocketKeepAliveEnabled {
		config.Net.KeepAlive = defaultKeepalive
	}

	config.Metadata.RefreshFrequency = o.KafkaConfig.TopicMetadataRefreshInterval.Duration()
	config.ClientID = fmt.Sprintf("%s-%s", o.ClientID, role)

	// Disable ApiVersionsRequest to avoid flooding the logs with ApiVersionsRequest errors from Sarama.
	// Error while sending ApiVersionsRequest V3 to broker.
	// See: https://github.com/IBM/sarama/blob/85f7d7b0cf3e3d4224df99c6b11f276c8fc49fd5/broker.go#L223-L251
	config.ApiVersionsRequest = false

	// These are globals, so we cannot append the publisher/subscriber name to them
	logger := o.Logger.With(slog.String(string(semconv.OTelScopeNameKey), "sarama"))

	sarama.Logger = &SaramaLoggerAdaptor{
		loggerFunc: logger.Info,
	}

	// NOTE: always set the sarama.DebugLogger otherwise the debug level logs are redirected to the sarama.Logger by default
	sarama.DebugLogger = &SaramaLoggerAdaptor{
		loggerFunc: logger.Debug,
	}

	switch o.KafkaConfig.SecurityProtocol {
	case "SASL_SSL":
		config.Net.SASL.Enable = true
		config.Net.SASL.Handshake = true

		config.Net.TLS.Enable = true
		// Sarama has issues with min version TLS 1.3, so let's use the defaults for now
		// remote error: tls: protocol version not supported
		config.Net.TLS.Config = &tls.Config{} // nosemgrep

		if err := o.applyCACertificates(config.Net.TLS.Config); err != nil {
			return nil, err
		}

		config.Net.SASL.User = o.KafkaConfig.SaslUsername
		config.Net.SASL.Password = o.KafkaConfig.SaslPassword
		// We rely on sarama to validate the SASL mechanism
		config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)

		switch o.KafkaConfig.SaslMechanisms {
		case sarama.SASLTypeSCRAMSHA256:
			config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			}
		case sarama.SASLTypeSCRAMSHA512:
			config.Net.SASL.Mechanism = sarama.SASLMechanism(o.KafkaConfig.SaslMechanisms)
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			}
		}
	case "SSL":
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{MinVersion: tls.VersionTLS12}

		if err := o.applyCACertificates(config.Net.TLS.Config); err != nil {
			return nil, err
		}
	}

	config.Producer.Retry.Max = 10
	config.Producer.MaxMessageBytes = 2000000 // 2MB
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
