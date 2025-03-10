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
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
	tracer := common.NewTracer(tracerProvider, commonMetadata)
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
	progressManagerConfiguration := conf.ProgressManager
	progressmanagerService, err := common.NewProgressManager(logger, progressManagerConfiguration)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	connector, err := common.NewStreamingConnector(ctx, aggregationConfiguration, v, logger, progressmanagerService)
	if err != nil {
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	meterService, err := common.NewMeterService(logger, client)
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
	v2 := common.ServerProvisionTopics(eventsConfiguration)
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
		ProvisionTopics:  v2,
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
	entitlement := common.NewEntitlementRegistry(logger, client, tracer, entitlementsConfiguration, connector, meterService, eventbusPublisher)
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
	featureConnector := common.NewFeatureConnector(logger, client, meterService)
	advancementStrategy := billingConfiguration.AdvancementStrategy
	billingService, err := common.BillingService(logger, client, service, adapter, billingConfiguration, customerService, featureConnector, meterService, connector, eventbusPublisher, advancementStrategy)
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
	v3 := common.NewNamespaceHandlers(namespaceHandler, connector)
	namespaceConfiguration := conf.Namespace
	manager, err := common.NewNamespaceManager(v3, namespaceConfiguration)
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
	tracer := common.NewTracer(tracerProvider, commonMetadata)
	collector, err := common.NewKafkaIngestCollector(kafkaIngestConfiguration, producer, namespacedTopicResolver, topicProvisioner, logger, tracer)
	if err != nil {
		cleanup6()
		cleanup5()
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
		return Application{}, nil, err
	}
	ingestCollector, cleanup7, err := common.NewIngestCollector(conf, collector, logger, meter, tracer)
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
	v4 := conf.Meters
	manageService, err := common.NewMeterManageService(ctx, client, logger, entitlement, manager, connector)
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
	v5 := common.NewMeterConfigInitializer(logger, v4, manageService, manager)
	metereventService := common.NewMeterEventService(connector)
	notificationConfiguration := conf.Notification
	v6 := conf.Svix
	notificationService, err := common.NewNotificationService(logger, client, notificationConfiguration, v6, featureConnector)
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
	portalConfiguration := conf.Portal
	portalService, err := common.NewPortalService(portalConfiguration)
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
	v7 := common.NewTelemetryRouterHook(meterProvider, tracerProvider)
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
	runtimeMetricsCollector, err := common.NewRuntimeMetricsCollector(meterProvider, telemetryConfig, logger)
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
	telemetryHandler := common.NewTelemetryHandler(metricsTelemetryConfig, health, runtimeMetricsCollector, logger)
	v8, cleanup8 := common.NewTelemetryServer(telemetryConfig, telemetryHandler)
	terminationConfig := conf.Termination
	terminationChecker, err := common.NewTerminationChecker(terminationConfig, health)
	if err != nil {
		cleanup8()
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
		GlobalInitializer:       globalInitializer,
		Migrator:                migrator,
		App:                     service,
		AppStripe:               appstripeService,
		AppSandboxProvisioner:   appSandboxProvisioner,
		Customer:                customerService,
		Billing:                 billingService,
		EntClient:               client,
		EventPublisher:          eventbusPublisher,
		EntitlementRegistry:     entitlement,
		FeatureConnector:        featureConnector,
		IngestCollector:         ingestCollector,
		KafkaProducer:           producer,
		KafkaMetrics:            metrics,
		Logger:                  logger,
		MetricMeter:             meter,
		MeterConfigInitializer:  v5,
		MeterManageService:      manageService,
		MeterEventService:       metereventService,
		NamespaceHandlers:       v3,
		NamespaceManager:        manager,
		Notification:            notificationService,
		Plan:                    planService,
		Portal:                  portalService,
		ProgressManager:         progressmanagerService,
		RouterHook:              v7,
		Secret:                  secretserviceService,
		Subscription:            subscriptionServiceWithWorkflow,
		StreamingConnector:      connector,
		TelemetryServer:         v8,
		TerminationChecker:      terminationChecker,
		RuntimeMetricsCollector: runtimeMetricsCollector,
		Tracer:                  tracer,
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

	App                     app.Service
	AppStripe               appstripe.Service
	AppSandboxProvisioner   common.AppSandboxProvisioner
	Customer                customer.Service
	Billing                 billing.Service
	EntClient               *db.Client
	EventPublisher          eventbus.Publisher
	EntitlementRegistry     *registry.Entitlement
	FeatureConnector        feature.FeatureConnector
	IngestCollector         ingest.Collector
	KafkaProducer           *kafka2.Producer
	KafkaMetrics            *metrics.Metrics
	Logger                  *slog.Logger
	MetricMeter             metric.Meter
	MeterConfigInitializer  common.MeterConfigInitializer
	MeterManageService      meter.ManageService
	MeterEventService       meterevent.Service
	NamespaceHandlers       []namespace.Handler
	NamespaceManager        *namespace.Manager
	Notification            notification.Service
	Plan                    plan.Service
	Portal                  portal.Service
	ProgressManager         progressmanager.Service
	RouterHook              func(chi.Router)
	Secret                  secret.Service
	Subscription            common.SubscriptionServiceWithWorkflow
	StreamingConnector      streaming.Connector
	TelemetryServer         common.TelemetryServer
	TerminationChecker      *common.TerminationChecker
	RuntimeMetricsCollector common.RuntimeMetricsCollector
	Tracer                  trace.Tracer
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "backend")
}
