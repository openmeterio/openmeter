package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestadapter"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func NewMeterRepository(meters []*models.Meter) *meter.InMemoryRepository {
	return meter.NewInMemoryRepository(slicesx.Map(meters, lo.FromPtr[models.Meter]))
}

func NewClickHouseStreamingConnector(
	conf config.AggregationConfiguration,
	clickHouse clickhouse.Conn,
	meterRepository meter.Repository,
	logger *slog.Logger,
) (*clickhouse_connector.ClickhouseConnector, error) {
	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		ClickHouse:           clickHouse,
		Database:             conf.ClickHouse.Database,
		Meters:               meterRepository,
		CreateOrReplaceMeter: conf.CreateOrReplaceMeter,
		PopulateMeter:        conf.PopulateMeter,
		Logger:               logger,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	return streamingConnector, nil
}

func NewNamespacedTopicResolver(config config.Configuration) (*topicresolver.NamespacedTopicResolver, error) {
	topicResolver, err := topicresolver.NewNamespacedTopicResolver(config.Ingest.Kafka.EventsTopicTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic name resolver: %w", err)
	}

	return topicResolver, nil
}

func NewKafkaIngestCollector(producer *kafka.Producer, topicResolver topicresolver.Resolver) (*kafkaingest.Collector, error) {
	collector, err := kafkaingest.NewCollector(
		producer,
		serializer.NewJSONSerializer(),
		topicResolver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka ingest: %w", err)
	}

	return collector, nil
}

func NewIngestCollector(
	conf config.Configuration,
	kafkaCollector *kafkaingest.Collector,
	logger *slog.Logger,
	meter metric.Meter,
) (ingest.Collector, func(), error) {
	collector, err := ingestadapter.WithMetrics(kafkaCollector, meter)
	if err != nil {
		return nil, nil, fmt.Errorf("init kafka ingest: %w", err)
	}

	if conf.Dedupe.Enabled {
		deduplicator, err := conf.Dedupe.NewDeduplicator()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize deduplicator: %w", err)
		}

		return ingest.DeduplicatingCollector{
				Collector:    collector,
				Deduplicator: deduplicator,
			}, func() {
				collector.Close()

				logger.Info("closing deduplicator")

				err := deduplicator.Close()
				if err != nil {
					logger.Error("failed to close deduplicator", "error", err)
				}
			}, nil
	}

	// Note: closing function is called by dedupe as well
	return collector, func() { collector.Close() }, nil
}

func NewKafkaNamespaceHandler(
	topicResolver topicresolver.Resolver,
	topicProvisioner pkgkafka.TopicProvisioner,
	conf config.Configuration,
) (*kafkaingest.NamespaceHandler, error) {
	return &kafkaingest.NamespaceHandler{
		TopicResolver:    topicResolver,
		TopicProvisioner: topicProvisioner,
		Partitions:       conf.Ingest.Kafka.Partitions,
	}, nil
}

func NewNamespaceHandlers(
	kafkaHandler *kafkaingest.NamespaceHandler,
	clickHouseHandler *clickhouse_connector.ClickhouseConnector,
) []namespace.Handler {
	return []namespace.Handler{
		kafkaHandler,
		clickHouseHandler,
	}
}

func NewNamespaceManager(
	handlers []namespace.Handler,
	conf config.NamespaceConfiguration,
) (*namespace.Manager, error) {
	manager, err := namespace.NewManager(namespace.ManagerConfig{
		Handlers:          handlers,
		DefaultNamespace:  conf.Default,
		DisableManagement: conf.DisableManagement,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace manager: %v", err)
	}

	return manager, nil
}

// TODO: create a separate file or package for each application instead

func NewServerPublisher(
	ctx context.Context,
	conf config.EventsConfiguration,
	options watermillkafka.PublisherOptions,
	logger *slog.Logger,
) (message.Publisher, func(), error) {
	if !conf.Enabled {
		return &noop.Publisher{}, func() {}, nil
	}

	return NewPublisher(ctx, options, logger)
}

// the sink-worker requires control over how the publisher is closed
func NewSinkWorkerPublisher(
	ctx context.Context,
	options watermillkafka.PublisherOptions,
	logger *slog.Logger,
) (message.Publisher, func(), error) {
	publisher, closer, err := NewPublisher(ctx, options, logger)

	closer = func() {}

	return publisher, closer, err
}

func NewFlushHandler(
	eventsConfig config.EventsConfiguration,
	sinkConfig config.SinkConfiguration,
	messagePublisher message.Publisher,
	eventPublisher eventbus.Publisher,
	logger *slog.Logger,
	meter metric.Meter,
) (flushhandler.FlushEventHandler, error) {
	if !eventsConfig.Enabled {
		return nil, nil
	}

	flushHandlerMux := flushhandler.NewFlushEventHandlers()

	// We should only close the producer once the ingest events are fully processed
	flushHandlerMux.OnDrainComplete(func() {
		logger.Info("shutting down kafka producer")
		if err := messagePublisher.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
		}
	})

	ingestNotificationHandler, err := ingestnotification.NewHandler(logger, meter, eventPublisher, ingestnotification.HandlerConfig{
		MaxEventsInBatch: sinkConfig.IngestNotifications.MaxEventsInBatch,
	})
	if err != nil {
		return nil, err
	}

	flushHandlerMux.AddHandler(ingestNotificationHandler)

	return flushHandlerMux, nil
}
