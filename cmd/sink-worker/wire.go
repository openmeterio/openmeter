//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-slog/otelslog"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type Application struct {
	app.GlobalInitializer

	Metadata app.Metadata

	MeterRepository  meter.Repository
	TelemetryServer  app.TelemetryServer
	FlushHandler     flushhandler.FlushEventHandler
	TopicProvisioner pkgkafka.TopicProvisioner

	Meter  metric.Meter
	Tracer trace.Tracer
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		metadata,
		app.Config,
		app.Framework,
		app.Telemetry,
		app.NewDefaultTextMapPropagator,
		app.KafkaTopic,
		app.SinkWorkerProvisionTopics,
		app.WatermillNoPublisher,
		newPublisher,
		app.OpenMeter,
		newFlushHandler,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	wire.Build(metadata, app.Config, app.Logger)

	return new(slog.Logger)
}

func metadata(conf config.Configuration) app.Metadata {
	return app.Metadata{
		ServiceName:       "openmeter",
		Version:           version,
		Environment:       conf.Environment,
		OpenTelemetryName: "openmeter.io/sink-worker",
	}
}

// TODO: use the primary logger
func NewLogger(conf config.Configuration, res *resource.Resource) *slog.Logger {
	logger := slog.New(otelslog.NewHandler(conf.Telemetry.Log.NewHandler(os.Stdout)))
	logger = otelslog.WithResource(logger, res)

	return logger
}

// the sink-worker requires control over how the publisher is closed
func newPublisher(
	ctx context.Context,
	options watermillkafka.PublisherOptions,
	logger *slog.Logger,
) (message.Publisher, func(), error) {
	publisher, closer, err := app.NewPublisher(ctx, options, logger)

	closer = func() {}

	return publisher, closer, err
}

func newFlushHandler(
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
