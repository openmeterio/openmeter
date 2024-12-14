// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
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
	eventsConfiguration := conf.Events
	balanceWorkerConfiguration := conf.BalanceWorker
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	brokerOptions := common.NewBrokerConfiguration(kafkaConfiguration, logTelemetryConfig, commonMetadata, logger, meter)
	subscriber, err := common.BalanceWorkerSubscriber(balanceWorkerConfiguration, brokerOptions)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v := common.BalanceWorkerProvisionTopics(balanceWorkerConfiguration)
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
	consumerConfiguration := balanceWorkerConfiguration.ConsumerConfiguration
	options := router.Options{
		Subscriber:  subscriber,
		Publisher:   publisher,
		Logger:      logger,
		MetricMeter: meter,
		Config:      consumerConfiguration,
	}
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
	aggregationConfiguration := conf.Aggregation
	clickHouseAggregationConfiguration := aggregationConfiguration.ClickHouse
	v2, err := common.NewClickHouse(clickHouseAggregationConfiguration)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v3 := conf.Meters
	inMemoryRepository := common.NewInMemoryRepository(v3)
	connector, err := common.NewStreamingConnector(ctx, aggregationConfiguration, v2, inMemoryRepository, logger)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	entitlementOptions := registrybuilder.EntitlementOptions{
		DatabaseClient:     client,
		StreamingConnector: connector,
		Logger:             logger,
		MeterRepository:    inMemoryRepository,
		Publisher:          eventbusPublisher,
	}
	entitlement := registrybuilder.GetEntitlementRegistry(entitlementOptions)
	balanceWorkerEntitlementRepo := common.NewBalanceWorkerEntitlementRepo(client)
	subjectResolver := common.BalanceWorkerSubjectResolver()
	workerOptions := common.NewBalanceWorkerOptions(eventsConfiguration, options, eventbusPublisher, entitlement, balanceWorkerEntitlementRepo, subjectResolver, logger)
	worker, err := common.NewBalanceWorker(workerOptions)
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
	v4, cleanup7 := common.NewTelemetryServer(telemetryConfig, telemetryHandler)
	group := common.BalanceWorkerGroup(ctx, worker, v4)
	runner := common.Runner{
		Group:  group,
		Logger: logger,
	}
	application := Application{
		GlobalInitializer: globalInitializer,
		Migrator:          migrator,
		Runner:            runner,
		Logger:            logger,
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
	common.Runner

	Logger *slog.Logger
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "balance-worker")
}
