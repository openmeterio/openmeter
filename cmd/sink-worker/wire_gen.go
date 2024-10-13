// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-slog/otelslog"
	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	kafka2 "github.com/openmeterio/openmeter/pkg/kafka"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"os"
)

// Injectors from wire.go:

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	telemetryConfig := conf.Telemetry
	metricsTelemetryConfig := telemetryConfig.Metrics
	appMetadata := metadata(conf)
	resource := app.NewTelemetryResource(appMetadata)
	meterProvider, cleanup, err := app.NewMeterProvider(ctx, metricsTelemetryConfig, resource, logger)
	if err != nil {
		return Application{}, nil, err
	}
	traceTelemetryConfig := telemetryConfig.Trace
	tracerProvider, cleanup2, err := app.NewTracerProvider(ctx, traceTelemetryConfig, resource, logger)
	if err != nil {
		cleanup()
		return Application{}, nil, err
	}
	textMapPropagator := app.NewDefaultTextMapPropagator()
	globalInitializer := app.GlobalInitializer{
		Logger:            logger,
		MeterProvider:     meterProvider,
		TracerProvider:    tracerProvider,
		TextMapPropagator: textMapPropagator,
	}
	v := conf.Meters
	inMemoryRepository := app.NewMeterRepository(v)
	health := app.NewHealthChecker(logger)
	telemetryHandler := app.NewTelemetryHandler(metricsTelemetryConfig, health)
	v2, cleanup3 := app.NewTelemetryServer(telemetryConfig, telemetryHandler)
	eventsConfiguration := conf.Events
	sinkConfiguration := conf.Sink
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	logTelemetryConfig := telemetryConfig.Log
	meter := app.NewMeter(meterProvider, appMetadata)
	brokerOptions := app.NewBrokerConfiguration(kafkaConfiguration, logTelemetryConfig, appMetadata, logger, meter)
	v3 := provisionTopics(eventsConfiguration)
	adminClient, err := app.NewKafkaAdminClient(kafkaConfiguration)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	topicProvisionerConfig := kafkaIngestConfiguration.TopicProvisionerConfig
	kafkaTopicProvisionerConfig := app.NewKafkaTopicProvisionerConfig(adminClient, logger, meter, topicProvisionerConfig)
	topicProvisioner, err := app.NewKafkaTopicProvisioner(kafkaTopicProvisionerConfig)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	publisherOptions := kafka.PublisherOptions{
		Broker:           brokerOptions,
		ProvisionTopics:  v3,
		TopicProvisioner: topicProvisioner,
	}
	publisher, cleanup4, err := newPublisher(ctx, publisherOptions, logger)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	eventbusPublisher, err := app.NewEventBusPublisher(publisher, eventsConfiguration, logger)
	if err != nil {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	flushEventHandler, err := newFlushHandler(eventsConfiguration, sinkConfiguration, publisher, eventbusPublisher, logger, meter)
	if err != nil {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	tracer := app.NewTracer(tracerProvider, appMetadata)
	application := Application{
		GlobalInitializer: globalInitializer,
		Metadata:          appMetadata,
		MeterRepository:   inMemoryRepository,
		TelemetryServer:   v2,
		FlushHandler:      flushEventHandler,
		TopicProvisioner:  topicProvisioner,
		Meter:             meter,
		Tracer:            tracer,
	}
	return application, func() {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
	}, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	telemetryConfig := conf.Telemetry
	logTelemetryConfig := telemetryConfig.Log
	appMetadata := metadata(conf)
	resource := app.NewTelemetryResource(appMetadata)
	logger := app.NewLogger(logTelemetryConfig, resource)
	return logger
}

// wire.go:

type Application struct {
	app.GlobalInitializer

	Metadata app.Metadata

	MeterRepository  meter.Repository
	TelemetryServer  app.TelemetryServer
	FlushHandler     flushhandler.FlushEventHandler
	TopicProvisioner kafka2.TopicProvisioner

	Meter  metric.Meter
	Tracer trace.Tracer
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

func provisionTopics(conf config.EventsConfiguration) []kafka2.TopicConfig {
	return []kafka2.TopicConfig{
		{
			Name:       conf.IngestEvents.Topic,
			Partitions: conf.IngestEvents.AutoProvision.Partitions,
		},
	}
}

// the sink-worker requires control over how the publisher is closed
func newPublisher(
	ctx context.Context,
	options kafka.PublisherOptions,
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
	logger *slog.Logger, meter2 metric.Meter,

) (flushhandler.FlushEventHandler, error) {
	if !eventsConfig.Enabled {
		return nil, nil
	}

	flushHandlerMux := flushhandler.NewFlushEventHandlers()

	flushHandlerMux.OnDrainComplete(func() {
		logger.Info("shutting down kafka producer")
		if err := messagePublisher.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
		}
	})

	ingestNotificationHandler, err := ingestnotification.NewHandler(logger, meter2, eventPublisher, ingestnotification.HandlerConfig{
		MaxEventsInBatch: sinkConfig.IngestNotifications.MaxEventsInBatch,
	})
	if err != nil {
		return nil, err
	}

	flushHandlerMux.AddHandler(ingestNotificationHandler)

	return flushHandlerMux, nil
}
