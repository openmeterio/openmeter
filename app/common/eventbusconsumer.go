package common

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/consumer"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func NewEventBusConsumerEnvironmentConfig(eventBusConfig config.ConsumerConfiguration,
	kafkaConsumerConfig pkgkafka.ConsumerConfig,
	meta Metadata,
	metricMeter metric.Meter,
	tracer trace.Tracer,
	logger *slog.Logger,
) (consumer.EnvironmentConfig, func(), error) {
	// misc
	if kafkaConsumerConfig.EnableAutoCommit {
		return consumer.EnvironmentConfig{}, nil, errors.New("enable auto commit is not supported for this consumer")
	}

	kafkaConsumer, shutdown, err := NewKafkaConsumer(kafkaConsumerConfig, logger)
	if err != nil {
		return consumer.EnvironmentConfig{}, nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// TODO[refactor]: The kafka producer should be initialized from the common config
	producerConfig, err := kafkaConsumerConfig.CommonConfigParams.AsConfigMap()
	if err != nil {
		return consumer.EnvironmentConfig{}, nil, fmt.Errorf("failed to create kafka producer config: %w", err)
	}
	_ = producerConfig.SetKey("client.id", meta.ServiceName)

	kafkaProducer, err := kafka.NewProducer(&producerConfig)
	if err != nil {
		return consumer.EnvironmentConfig{}, nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	go pkgkafka.ConsumeLogChannel(kafkaProducer, logger.WithGroup("kafka").WithGroup("producer"))

	return consumer.EnvironmentConfig{
			Consumer: kafkaConsumer,
			Producer: kafkaProducer,

			ConsumerConfig: eventBusConfig,

			Logger:      logger,
			MetricMeter: metricMeter,
			Tracer:      tracer,
		}, func() {
			shutdown()
			kafkaProducer.Close()
		}, nil
}
