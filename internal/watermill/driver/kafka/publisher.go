package kafka

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/watermill/driver/kafka/metrics"
)

const (
	defaultMeterPrefix = "sarama.publisher."
)

type PublisherOptions struct {
	KafkaConfig     config.KafkaConfiguration
	ProvisionTopics []AutoProvisionTopic
	ClientID        string
	Logger          *slog.Logger
	MetricMeter     otelmetric.Meter
	MeterPrefix     string
}

func (o *PublisherOptions) Validate() error {
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

func NewPublisher(in PublisherOptions) (*kafka.Publisher, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	if in.MeterPrefix == "" {
		in.MeterPrefix = defaultMeterPrefix
	}

	wmConfig := kafka.PublisherConfig{
		Brokers:               []string{in.KafkaConfig.Broker},
		OverwriteSaramaConfig: sarama.NewConfig(),
		Marshaler:             marshalerWithPartitionKey{},
		OTELEnabled:           true, // This relies on the global trace provider
	}

	wmConfig.OverwriteSaramaConfig.Metadata.RefreshFrequency = in.KafkaConfig.TopicMetadataRefreshInterval.Duration()
	wmConfig.OverwriteSaramaConfig.ClientID = "openmeter/balance-worker"

	switch in.KafkaConfig.SecurityProtocol {
	case "SASL_SSL":
		wmConfig.OverwriteSaramaConfig.Net.SASL.Enable = true
		wmConfig.OverwriteSaramaConfig.Net.SASL.User = in.KafkaConfig.SaslUsername
		wmConfig.OverwriteSaramaConfig.Net.SASL.Password = in.KafkaConfig.SaslPassword
		wmConfig.OverwriteSaramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(in.KafkaConfig.SecurityProtocol)
		wmConfig.OverwriteSaramaConfig.Net.TLS.Enable = true
		wmConfig.OverwriteSaramaConfig.Net.TLS.Config = &tls.Config{}
	default:
	}

	// Producer specific settings
	wmConfig.OverwriteSaramaConfig.Producer.Return.Successes = true

	meterRegistry, err := metrics.NewRegistry(metrics.NewRegistryOptions{
		MetricMeter:     in.MetricMeter,
		NameTransformFn: metrics.MetricAddNamePrefix(in.MeterPrefix),
		ErrorHandler:    metrics.LoggingErrorHandler(in.Logger),
	})
	if err != nil {
		return nil, err
	}

	wmConfig.OverwriteSaramaConfig.MetricRegistry = meterRegistry

	if err := wmConfig.Validate(); err != nil {
		return nil, err
	}

	if err := provisionTopics(in.KafkaConfig.Broker, wmConfig.OverwriteSaramaConfig, in.ProvisionTopics); err != nil {
		return nil, err
	}

	return kafka.NewPublisher(wmConfig, watermill.NewSlogLogger(in.Logger))
}
