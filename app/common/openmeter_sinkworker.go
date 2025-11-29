package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var Sink = wire.NewSet(
	NewIngestNotificationHandler,
	NewFlushHandlers,
	NewFlushHandlerManager,
	NewSinkWorkerPublisher,
	NewSinkKafkaConsumer,
	NewSinkDeduplicator,
	NewSinkStorage,
	NewSink,
)

// the sink-worker requires control over how the publisher is closed
func NewSinkWorkerPublisher(
	ctx context.Context,
	options watermillkafka.PublisherOptions,
	logger *slog.Logger,
) (message.Publisher, func(), error) {
	publisher, _, err := NewPublisher(ctx, options, logger)

	return publisher, func() {}, err
}

type IngestNotificationHandler flushhandler.FlushEventHandler

func NewIngestNotificationHandler(
	sinkConfig config.SinkConfiguration,
	eventPublisher eventbus.Publisher,
	meter metric.Meter,
	logger *slog.Logger,
) (IngestNotificationHandler, error) {
	handler, err := ingestnotification.NewHandler(logger, meter, eventPublisher, ingestnotification.HandlerConfig{
		MaxEventsInBatch: sinkConfig.IngestNotifications.MaxEventsInBatch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ingest notification handler: %w", err)
	}

	return handler, nil
}

func NewFlushHandlers(
	ingest IngestNotificationHandler,
) []flushhandler.FlushEventHandler {
	return []flushhandler.FlushEventHandler{ingest}
}

func NewFlushHandlerManager(
	sinkConfig config.SinkConfiguration,
	messagePublisher message.Publisher,
	logger *slog.Logger,
	handlers []flushhandler.FlushEventHandler,
) (flushhandler.FlushEventHandler, func(), error) {
	flushHandlerMux := flushhandler.NewFlushEventHandlers()

	// We should only close the producer once the ingest events are fully processed
	flushHandlerMux.OnDrainComplete(func() {
		logger.Info("shutting down kafka producer")
		if err := messagePublisher.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
		}
	})

	for _, handler := range handlers {
		if handler != nil {
			flushHandlerMux.AddHandler(handler)
		}
	}

	closer := func() {
		logger.Info("shutting down flush success handlers")

		if err := flushHandlerMux.Close(); err != nil {
			logger.Error("failed to close flush success handler", slog.String("err", err.Error()))
		}

		drainCtx, cancel := context.WithTimeout(context.Background(), sinkConfig.DrainTimeout)
		defer cancel()

		if err := flushHandlerMux.WaitForDrain(drainCtx); err != nil {
			logger.Error("failed to drain flush success handlers", slog.String("err", err.Error()))
		}
	}

	return flushHandlerMux, closer, nil
}

func SinkWorkerProvisionTopics(conf config.EventsConfiguration) []pkgkafka.TopicConfig {
	return []pkgkafka.TopicConfig{
		{
			Name:       conf.IngestEvents.Topic,
			Partitions: conf.IngestEvents.AutoProvision.Partitions,
		},
	}
}

func NewSinkStorage(
	streaming streaming.Connector,
) (sink.Storage, error) {
	return sink.NewClickhouseStorage(sink.ClickHouseStorageConfig{
		Streaming: streaming,
	})
}

func NewSinkDeduplicator(conf config.SinkConfiguration, logger *slog.Logger) (dedupe.Deduplicator, func(), error) {
	closer := func() {}

	if conf.Dedupe.Enabled {
		deduplicator, err := conf.Dedupe.NewDeduplicator()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize deduplicator: %w", err)
		}

		closer = func() {
			err = deduplicator.Close()
			if err != nil {
				logger.Error("failed to close sink deduplicator", slog.String("err", err.Error()))
			}
		}

		return deduplicator, closer, nil
	}

	return nil, closer, nil
}

type SinkKafkaConsumer = kafka.Consumer

func NewSinkKafkaConsumer(conf config.SinkConfiguration, logger *slog.Logger) (*kafka.Consumer, func(), error) {
	consumerConfig := conf.Kafka.AsConsumerConfig()

	// Override the following Kafka consumer configuration parameters with hardcoded values
	// as the Sink implementation relies on these to be set to a specific value.
	consumerConfig.EnableAutoCommit = true
	consumerConfig.EnableAutoOffsetStore = false
	// Used when offset retention resets the offset. In this case we want to consume from the latest offset
	// as everything before should be already processed.
	consumerConfig.AutoOffsetReset = pkgkafka.AutoOffsetResetLatest

	return NewKafkaConsumer(consumerConfig, logger)
}

func NewSink(
	conf config.SinkConfiguration,
	logger *slog.Logger,
	metricMeter metric.Meter,
	tracer trace.Tracer,
	kafkaConsumer *SinkKafkaConsumer,
	sinkStorage sink.Storage,
	deduplicator dedupe.Deduplicator,
	meterService meter.Service,
	topicResolver topicresolver.Resolver,
	flushHandler flushhandler.FlushEventHandler,
) (*sink.Sink, func(), error) {
	s, err := sink.NewSink(sink.SinkConfig{
		Logger:                  logger,
		Tracer:                  tracer,
		MetricMeter:             metricMeter,
		Storage:                 sinkStorage,
		Deduplicator:            deduplicator,
		Consumer:                kafkaConsumer,
		MinCommitCount:          conf.MinCommitCount,
		MaxCommitWait:           conf.MaxCommitWait,
		MaxPollTimeout:          conf.MaxPollTimeout,
		NamespaceRefetch:        conf.NamespaceRefetch,
		NamespaceRefetchTimeout: conf.NamespaceRefetchTimeout,
		NamespaceTopicRegexp:    conf.NamespaceTopicRegexp,
		FlushEventHandler:       flushHandler,
		FlushSuccessTimeout:     conf.FlushSuccessTimeout,
		DrainTimeout:            conf.DrainTimeout,
		TopicResolver:           topicResolver,
		MeterRefetchInterval:    conf.MeterRefetchInterval,
		MeterService:            meterService,
		LogDroppedEvents:        conf.LogDroppedEvents,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize sink: %w", err)
	}

	closer := func() {
		if err = s.Close(); err != nil {
			logger.Error("failed to close sink worker", slog.String("err", err.Error()))
		}
	}

	return s, closer, nil
}
