package app

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

func NewKafkaTopicProvisioner(conf config.IngestConfiguration, logger *slog.Logger, meter metric.Meter) (pkgkafka.TopicProvisioner, error) {
	kafkaConfigMap := conf.Kafka.CreateKafkaConfig()
	// NOTE(chrisgacsal): remove 'go.logs.channel.enable' configuration parameter as it is not supported by AdminClient
	// and initializing the client fails if this parameter is set.
	delete(kafkaConfigMap, "go.logs.channel.enable")

	adminClient, err := kafka.NewAdminClient(&kafkaConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka admin client: %w", err)
	}

	topicProvisioner, err := pkgkafka.NewTopicProvisioner(pkgkafka.TopicProvisionerConfig{
		AdminClient: adminClient,
		Logger:      logger,
		Meter:       meter,
		CacheSize:   conf.Kafka.CacheSize,
		CacheTTL:    conf.Kafka.CacheTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize topic provisioner: %w", err)
	}

	return topicProvisioner, nil
}
