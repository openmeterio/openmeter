// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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
	billingConfiguration := conf.Billing
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	brokerOptions := common.NewBrokerConfiguration(kafkaConfiguration, logTelemetryConfig, commonMetadata, logger, meter)
	subscriber, err := common.BillingWorkerSubscriber(billingConfiguration, brokerOptions)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v := common.BillingWorkerProvisionTopics(billingConfiguration)
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
	billingWorkerConfiguration := billingConfiguration.Worker
	consumerConfiguration := billingWorkerConfiguration.ConsumerConfiguration
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
	appsConfiguration := conf.Apps
	service, err := common.NewAppService(logger, client, appsConfiguration)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	entitlementsConfiguration := conf.Entitlements
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
	entitlement := common.NewEntitlementRegistry(logger, client, entitlementsConfiguration, connector, inMemoryRepository, eventbusPublisher)
	customerService, err := common.NewCustomerService(logger, client, entitlement)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	secretserviceService, err := common.NewUnsafeSecretService(logger, client)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	appstripeService, err := common.NewAppStripeService(logger, client, appsConfiguration, service, customerService, secretserviceService)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	adapter, err := common.BillingAdapter(logger, client)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	featureConnector := common.NewFeatureConnector(logger, client, inMemoryRepository)
	billingService, err := common.BillingService(logger, client, service, appstripeService, adapter, billingConfiguration, customerService, featureConnector, inMemoryRepository, connector, eventbusPublisher)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	productCatalogConfiguration := conf.ProductCatalog
	planService, err := common.NewPlanService(logger, client, productCatalogConfiguration, featureConnector)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	validator, err := common.BillingSubscriptionValidator(billingService, billingConfiguration)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	subscriptionServiceWithWorkflow, err := common.NewSubscriptionServices(logger, client, productCatalogConfiguration, entitlementsConfiguration, featureConnector, entitlement, customerService, planService, eventbusPublisher, validator)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	workerOptions := common.NewBillingWorkerOptions(eventsConfiguration, options, eventbusPublisher, billingService, adapter, subscriptionServiceWithWorkflow, logger)
	worker, err := common.NewBillingWorker(workerOptions)
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
	group := common.BillingWorkerGroup(ctx, worker, v4)
	runner := common.Runner{
		Group:  group,
		Logger: logger,
	}
	namespacedTopicResolver, err := common.NewNamespacedTopicResolver(kafkaIngestConfiguration)
	if err != nil {
		cleanup7()
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	namespaceHandler, err := common.NewKafkaNamespaceHandler(namespacedTopicResolver, topicProvisioner, kafkaIngestConfiguration)
	if err != nil {
		cleanup7()
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v5 := common.NewNamespaceHandlers(namespaceHandler, connector)
	namespaceConfiguration := conf.Namespace
	manager, err := common.NewNamespaceManager(v5, namespaceConfiguration)
	if err != nil {
		cleanup7()
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	appSandboxProvisioner, err := common.NewAppSandboxProvisioner(ctx, logger, appsConfiguration, service, manager)
	if err != nil {
		cleanup7()
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	application := Application{
		GlobalInitializer:     globalInitializer,
		Migrator:              migrator,
		Runner:                runner,
		App:                   service,
		AppStripe:             appstripeService,
		AppSandboxProvisioner: appSandboxProvisioner,
		Logger:                logger,
		Meter:                 inMemoryRepository,
		Streaming:             connector,
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

	App                   app.Service
	AppStripe             appstripe.Service
	AppSandboxProvisioner common.AppSandboxProvisioner
	Logger                *slog.Logger
	Meter                 meter.Repository
	Streaming             streaming.Connector
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "billing-worker")
}
