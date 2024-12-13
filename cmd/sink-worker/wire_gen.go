// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/go-slog/otelslog"
	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	kafka2 "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/samber/slog-multi"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"os"
)

// Injectors from wire.go:

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	telemetryConfig := conf.Telemetry
	logTelemetryConfig := telemetryConfig.Log
	commonMetadata := metadata(conf)
	resource := common.NewTelemetryResource(commonMetadata)
	loggerProvider, cleanup, err := common.NewLoggerProvider(ctx, logTelemetryConfig, resource)
	if err != nil {
		return Application{}, nil, err
	}
	logger := common.NewLogger(logTelemetryConfig, resource, loggerProvider, commonMetadata)
	metricsTelemetryConfig := telemetryConfig.Metrics
	meterProvider, cleanup2, err := common.NewMeterProvider(ctx, metricsTelemetryConfig, resource, logger)
	if err != nil {
		cleanup()
		return Application{}, nil, err
	}
	traceTelemetryConfig := telemetryConfig.Trace
	tracerProvider, cleanup3, err := common.NewTracerProvider(ctx, traceTelemetryConfig, resource, logger)
	if err != nil {
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	textMapPropagator := common.NewDefaultTextMapPropagator()
	globalInitializer := common.GlobalInitializer{
		Logger:            logger,
		MeterProvider:     meterProvider,
		TracerProvider:    tracerProvider,
		TextMapPropagator: textMapPropagator,
	}
	v := conf.Meters
	inMemoryRepository := common.NewMeterRepository(v)
	health := common.NewHealthChecker(logger)
	telemetryHandler := common.NewTelemetryHandler(metricsTelemetryConfig, health)
	v2, cleanup4 := common.NewTelemetryServer(telemetryConfig, telemetryHandler)
	eventsConfiguration := conf.Events
	sinkConfiguration := conf.Sink
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	meter := common.NewMeter(meterProvider, commonMetadata)
	brokerOptions := common.NewBrokerConfiguration(kafkaConfiguration, logTelemetryConfig, commonMetadata, logger, meter)
	v3 := common.SinkWorkerProvisionTopics(eventsConfiguration)
	adminClient, err := common.NewKafkaAdminClient(kafkaConfiguration)
	if err != nil {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	topicProvisionerConfig := kafkaIngestConfiguration.TopicProvisionerConfig
	kafkaTopicProvisionerConfig := common.NewKafkaTopicProvisionerConfig(adminClient, logger, meter, topicProvisionerConfig)
	topicProvisioner, err := common.NewKafkaTopicProvisioner(kafkaTopicProvisionerConfig)
	if err != nil {
		cleanup4()
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
	publisher, cleanup5, err := common.NewSinkWorkerPublisher(ctx, publisherOptions, logger)
	if err != nil {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	eventbusPublisher, err := common.NewEventBusPublisher(publisher, eventsConfiguration, logger)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	flushEventHandler, err := common.NewFlushHandler(eventsConfiguration, sinkConfiguration, publisher, eventbusPublisher, logger, meter)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	tracer := common.NewTracer(tracerProvider, commonMetadata)
	application := Application{
		GlobalInitializer: globalInitializer,
		Metadata:          commonMetadata,
		MeterRepository:   inMemoryRepository,
		TelemetryServer:   v2,
		FlushHandler:      flushEventHandler,
		TopicProvisioner:  topicProvisioner,
		Logger:            logger,
		Meter:             meter,
		Tracer:            tracer,
	}
	return application, func() {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
	}, nil
}

// wire.go:

type Application struct {
	common.GlobalInitializer

	Metadata common.Metadata

	MeterRepository  meter.Repository
	TelemetryServer  common.TelemetryServer
	FlushHandler     flushhandler.FlushEventHandler
	TopicProvisioner kafka2.TopicProvisioner

	Logger *slog.Logger
	Meter  metric.Meter
	Tracer trace.Tracer
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "sink-worker")
}

// TODO: use the primary logger
func NewLogger(conf config.LogTelemetryConfig, res *resource.Resource) *slog.Logger {
	return slog.New(slogmulti.Pipe(otelslog.ResourceMiddleware(res), otelslog.NewHandler).Handler(conf.NewHandler(os.Stdout)))
}
