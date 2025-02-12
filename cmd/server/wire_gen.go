// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	kafka2 "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics"
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
	appsConfiguration := conf.Apps
	service, err := common.NewAppService(logger, client, appsConfiguration)
	if err != nil {
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
	v, err := common.NewClickHouse(clickHouseAggregationConfiguration)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v2 := conf.Meters
	inMemoryRepository := common.NewInMemoryRepository(v2)
	connector, err := common.NewStreamingConnector(ctx, aggregationConfiguration, v, inMemoryRepository, logger)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	ingestConfiguration := conf.Ingest
	kafkaIngestConfiguration := ingestConfiguration.Kafka
	kafkaConfiguration := kafkaIngestConfiguration.KafkaConfiguration
	brokerOptions := common.NewBrokerConfiguration(kafkaConfiguration, commonMetadata, logger, meter)
	eventsConfiguration := conf.Events
	v3 := common.ServerProvisionTopics(eventsConfiguration)
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
		ProvisionTopics:  v3,
		TopicProvisioner: topicProvisioner,
	}
	publisher, cleanup6, err := common.NewServerPublisher(ctx, publisherOptions, logger)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
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
	billingConfiguration := conf.Billing
	featureConnector := common.NewFeatureConnector(logger, client, inMemoryRepository)
	advancementStrategy := billingConfiguration.AdvancementStrategy
	billingService, err := common.BillingService(logger, client, service, adapter, billingConfiguration, customerService, featureConnector, inMemoryRepository, connector, eventbusPublisher, advancementStrategy)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	appstripeService, err := common.NewAppStripeService(logger, client, appsConfiguration, service, customerService, secretserviceService, billingService)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	namespacedTopicResolver, err := common.NewNamespacedTopicResolver(kafkaIngestConfiguration)
	if err != nil {
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
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	v4 := common.NewNamespaceHandlers(namespaceHandler, connector)
	namespaceConfiguration := conf.Namespace
	manager, err := common.NewNamespaceManager(v4, namespaceConfiguration)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	appSandboxProvisioner, err := common.NewAppSandboxProvisioner(ctx, logger, appsConfiguration, service, manager, billingService)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	producer, err := common.NewKafkaProducer(kafkaIngestConfiguration, logger)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	collector, err := common.NewKafkaIngestCollector(kafkaIngestConfiguration, producer, namespacedTopicResolver, topicProvisioner)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	ingestCollector, cleanup7, err := common.NewIngestCollector(conf, collector, logger, meter)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	metrics, err := common.NewKafkaMetrics(meter)
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
	notificationConfiguration := conf.Notification
	v5 := conf.Svix
	notificationService, err := common.NewNotificationService(logger, client, notificationConfiguration, v5, featureConnector)
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
	productCatalogConfiguration := conf.ProductCatalog
	planService, err := common.NewPlanService(logger, client, productCatalogConfiguration, featureConnector)
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
	v6 := common.NewTelemetryRouterHook(meterProvider, tracerProvider)
	validator, err := common.BillingSubscriptionValidator(billingService, billingConfiguration)
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
	subscriptionServiceWithWorkflow, err := common.NewSubscriptionServices(logger, client, productCatalogConfiguration, entitlementsConfiguration, featureConnector, entitlement, customerService, planService, eventbusPublisher, validator)
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
	health := common.NewHealthChecker(logger)
	telemetryHandler := common.NewTelemetryHandler(metricsTelemetryConfig, health)
	v7, cleanup8 := common.NewTelemetryServer(telemetryConfig, telemetryHandler)
	application := Application{
		GlobalInitializer:     globalInitializer,
		Migrator:              migrator,
		App:                   service,
		AppStripe:             appstripeService,
		AppSandboxProvisioner: appSandboxProvisioner,
		Customer:              customerService,
		Billing:               billingService,
		EntClient:             client,
		EventPublisher:        eventbusPublisher,
		EntitlementRegistry:   entitlement,
		FeatureConnector:      featureConnector,
		IngestCollector:       ingestCollector,
		KafkaProducer:         producer,
		KafkaMetrics:          metrics,
		Logger:                logger,
		MeterRepository:       inMemoryRepository,
		NamespaceHandlers:     v4,
		NamespaceManager:      manager,
		Notification:          notificationService,
		Meter:                 meter,
		Plan:                  planService,
		RouterHook:            v6,
		Secret:                secretserviceService,
		Subscription:          subscriptionServiceWithWorkflow,
		StreamingConnector:    connector,
		TelemetryServer:       v7,
	}
	return application, func() {
		cleanup8()
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

	App                   app.Service
	AppStripe             appstripe.Service
	AppSandboxProvisioner common.AppSandboxProvisioner
	Customer              customer.Service
	Billing               billing.Service
	EntClient             *db.Client
	EventPublisher        eventbus.Publisher
	EntitlementRegistry   *registry.Entitlement
	FeatureConnector      feature.FeatureConnector
	IngestCollector       ingest.Collector
	KafkaProducer         *kafka2.Producer
	KafkaMetrics          *metrics.Metrics
	Logger                *slog.Logger
	MeterRepository       meter.Repository
	NamespaceHandlers     []namespace.Handler
	NamespaceManager      *namespace.Manager
	Notification          notification.Service
	Meter                 metric.Meter
	Plan                  plan.Service
	RouterHook            func(chi.Router)
	Secret                secret.Service
	Subscription          common.SubscriptionServiceWithWorkflow
	StreamingConnector    streaming.Connector
	TelemetryServer       common.TelemetryServer
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "backend")
}
