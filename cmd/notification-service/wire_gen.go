// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
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
	postgresConfig := conf.Postgres
	meter := common.NewMeter(meterProvider, commonMetadata)
	driver, cleanup4, err := common.NewPostgresDriver(ctx, postgresConfig, meterProvider, meter, tracerProvider, logger)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	db := common.NewDB(driver)
	entPostgresDriver, cleanup5 := common.NewEntPostgresDriver(db, logger)
	client := common.NewEntClient(entPostgresDriver)
	migrator := common.Migrator{
		Config: postgresConfig,
		Client: client,
		Logger: logger,
	}
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	brokerOptions := common.NewBrokerConfiguration(kafkaConfiguration, logTelemetryConfig, commonMetadata, logger, meter)
	notificationConfiguration := conf.Notification
	v := common.NotificationServiceProvisionTopics(notificationConfiguration)
	adminClient, err := common.NewKafkaAdminClient(kafkaConfiguration)
	if err != nil {
		cleanup5()
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
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	publisherOptions := kafka.PublisherOptions{
		Broker:           brokerOptions,
		ProvisionTopics:  v,
		TopicProvisioner: topicProvisioner,
	}
	publisher, cleanup6, err := common.NewPublisher(ctx, publisherOptions, logger)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	eventsConfiguration := conf.Events
	eventbusPublisher, err := common.NewEventBusPublisher(publisher, eventsConfiguration, logger)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v2 := conf.Meters
	inMemoryRepository := common.NewMeterRepository(v2)
	featureConnector := common.NewFeatureConnector(logger, client, inMemoryRepository)
	v3 := conf.Svix
	service, err := common.NewNotificationService(logger, client, notificationConfiguration, v3, featureConnector)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	aggregationConfiguration := conf.Aggregation
	clickHouseAggregationConfiguration := aggregationConfiguration.ClickHouse
	v4, err := common.NewClickHouse(clickHouseAggregationConfiguration)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	connector, err := common.NewStreamingConnector(ctx, aggregationConfiguration, v4, inMemoryRepository, logger)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	health := common.NewHealthChecker(logger)
	telemetryHandler := common.NewTelemetryHandler(metricsTelemetryConfig, health)
	v5, cleanup7 := common.NewTelemetryServer(telemetryConfig, telemetryHandler)
	application := Application{
		GlobalInitializer:  globalInitializer,
		Migrator:           migrator,
		BrokerOptions:      brokerOptions,
		EventPublisher:     eventbusPublisher,
		EntClient:          client,
		FeatureConnector:   featureConnector,
		Logger:             logger,
		MessagePublisher:   publisher,
		Meter:              meter,
		Metadata:           commonMetadata,
		MeterRepository:    inMemoryRepository,
		Notification:       service,
		StreamingConnector: connector,
		TelemetryServer:    v5,
	}
	return application, func() {
		cleanup7()
		cleanup6()
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
	common.Migrator

	BrokerOptions      kafka.BrokerOptions
	EventPublisher     eventbus.Publisher
	EntClient          *db.Client
	FeatureConnector   feature.FeatureConnector
	Logger             *slog.Logger
	MessagePublisher   message.Publisher
	Meter              metric.Meter
	Metadata           common.Metadata
	MeterRepository    meter.Repository
	Notification       notification.Service
	StreamingConnector streaming.Connector
	TelemetryServer    common.TelemetryServer
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "notification-worker")
}
