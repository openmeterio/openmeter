package common

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
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

func NewFlushHandler(
	eventsConfig config.EventsConfiguration,
	sinkConfig config.SinkConfiguration,
	messagePublisher message.Publisher,
	eventPublisher eventbus.Publisher,
	logger *slog.Logger,
	meter metric.Meter,
) (flushhandler.FlushEventHandler, error) {
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

func SinkWorkerProvisionTopics(conf config.EventsConfiguration) []pkgkafka.TopicConfig {
	return []pkgkafka.TopicConfig{
		{
			Name:       conf.IngestEvents.Topic,
			Partitions: conf.IngestEvents.AutoProvision.Partitions,
		},
	}
}
