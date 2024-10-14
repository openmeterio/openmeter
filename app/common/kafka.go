package common

import (
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

// TODO: use ingest config directly?
// TODO: use kafka config directly?
// TODO: add closer function?
func NewKafkaProducer(conf config.Configuration, logger *slog.Logger) (*kafka.Producer, error) {
	// Initialize Kafka Admin Client
	kafkaConfig := conf.Ingest.Kafka.CreateKafkaConfig()

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(&kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka producer: %w", err)
	}

	// TODO: move out of this function?
	go pkgkafka.ConsumeLogChannel(producer, logger.WithGroup("kafka").WithGroup("producer"))

	// TODO: remove?
	logger.Debug("connected to Kafka")

	return producer, nil
}

func NewKafkaMetrics(meter metric.Meter) (*kafkametrics.Metrics, error) {
	metrics, err := kafkametrics.New(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client metrics: %w", err)
	}

	return metrics, nil
}

func NewKafkaAdminClient(conf config.KafkaConfiguration) (*kafka.AdminClient, error) {
	kafkaConfigMap := conf.CreateKafkaConfig()
	// NOTE(chrisgacsal): remove 'go.logs.channel.enable' configuration parameter as it is not supported by AdminClient
	// and initializing the client fails if this parameter is set.
	delete(kafkaConfigMap, "go.logs.channel.enable")

	adminClient, err := kafka.NewAdminClient(&kafkaConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka admin client: %w", err)
	}

	return adminClient, nil
}

// TODO: fill struct fields automatically?
func NewKafkaTopicProvisionerConfig(
	adminClient *kafka.AdminClient,
	logger *slog.Logger,
	meter metric.Meter,
	settings config.TopicProvisionerConfig,
) pkgkafka.TopicProvisionerConfig {
	return pkgkafka.TopicProvisionerConfig{
		AdminClient: adminClient,
		Logger:      logger,
		Meter:       meter,
		CacheSize:   settings.CacheSize,
		CacheTTL:    settings.CacheTTL,
	}
}

// TODO: do we need a separate constructor for the sake of a custom error message?
func NewKafkaTopicProvisioner(conf pkgkafka.TopicProvisionerConfig) (pkgkafka.TopicProvisioner, error) {
	topicProvisioner, err := pkgkafka.NewTopicProvisioner(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize topic provisioner: %w", err)
	}

	return topicProvisioner, nil
}
