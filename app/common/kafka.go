package common

import (
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

var Kafka = wire.NewSet(
	NewKafkaProducer,
	NewKafkaMetrics,

	NewKafkaTopicProvisioner,
)

var KafkaIngest = wire.NewSet(
	NewKafkaIngestNamespaceHandler,
)

var KafkaNamespaceResolver = wire.NewSet(
	NewNamespacedTopicResolver,
	wire.Bind(new(topicresolver.Resolver), new(*topicresolver.NamespacedTopicResolver)),
)

// TODO: add closer function?
func NewKafkaProducer(conf config.KafkaIngestConfiguration, logger *slog.Logger, meta Metadata) (*kafka.Producer, error) {
	kafkaConfig := conf.CreateKafkaConfig()
	_ = kafkaConfig.SetKey("client.id", meta.ServiceName)

	producer, err := kafka.NewProducer(&kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka producer: %w", err)
	}

	go pkgkafka.ConsumeLogChannel(producer, logger.WithGroup("kafka").WithGroup("producer"))

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

	// NOTE(chrisgacsal): disable collecting statistics as data is collected in an internal queue which needs to be polled,
	// but the AdminClient does not expose interface for that.
	delete(kafkaConfigMap, "statistics.interval.ms")

	adminClient, err := kafka.NewAdminClient(&kafkaConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka admin client: %w", err)
	}

	return adminClient, nil
}

func NewKafkaTopicProvisioner(
	kafkaConfig config.KafkaConfiguration,
	settings config.TopicProvisionerConfig,
	logger *slog.Logger,
	meter metric.Meter,
) (pkgkafka.TopicProvisioner, error) {
	if !settings.Enabled {
		return &pkgkafka.TopicProvisionerNoop{}, nil
	}

	adminClient, err := NewKafkaAdminClient(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka admin client: %w", err)
	}

	provisionerConfig := pkgkafka.TopicProvisionerConfig{
		AdminClient:     adminClient,
		Logger:          logger,
		Meter:           meter,
		CacheSize:       settings.CacheSize,
		CacheTTL:        settings.CacheTTL,
		ProtectedTopics: settings.ProtectedTopics,
	}

	topicProvisioner, err := pkgkafka.NewTopicProvisioner(provisionerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize topic provisioner: %w", err)
	}

	return topicProvisioner, nil
}

func NewNoopKafkaTopicProvisioner() pkgkafka.TopicProvisioner {
	return &pkgkafka.TopicProvisionerNoop{}
}

func NewNamespacedTopicResolver(config config.KafkaIngestConfiguration) (*topicresolver.NamespacedTopicResolver, error) {
	topicResolver, err := topicresolver.NewNamespacedTopicResolver(config.EventsTopicTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic name resolver: %w", err)
	}

	return topicResolver, nil
}

func NewKafkaIngestNamespaceHandler(
	topicResolver topicresolver.Resolver,
	topicProvisioner pkgkafka.TopicProvisioner,
	ingestConfig config.KafkaIngestConfiguration,
) (*kafkaingest.NamespaceHandler, error) {
	handler := &kafkaingest.NamespaceHandler{
		TopicResolver:    topicResolver,
		TopicProvisioner: topicProvisioner,
		Partitions:       ingestConfig.Partitions,
		DeletionEnabled:  ingestConfig.NamespaceDeletionEnabled,
	}

	return handler, nil
}

func NewKafkaConsumer(conf pkgkafka.ConsumerConfig, logger *slog.Logger) (*kafka.Consumer, func(), error) {
	if err := conf.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid Kafka consumer configuration: %w", err)
	}

	consumerConfigMap, err := conf.AsConfigMap()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Kafka consumer configuration map: %w", err)
	}

	consumer, err := kafka.NewConsumer(&consumerConfigMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}

	kLogger := logger.WithGroup("kafka").WithGroup("consumer").With(
		"group.id", conf.ConsumerGroupID,
		"group.instance.id", conf.ConsumerGroupInstanceID,
		"client.id", conf.ClientID,
	)

	// Enable Kafka client logging
	// TODO: refactor ConsumeLogChannel to allow graceful shutdown
	go pkgkafka.ConsumeLogChannel(consumer, kLogger)

	closer := func() {
		if err = consumer.Close(); err != nil {
			kLogger.Error("failed to close Kafka consumer", slog.String("err", err.Error()))
		}
	}

	return consumer, closer, nil
}
