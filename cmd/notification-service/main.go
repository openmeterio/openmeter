package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/contextx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/driver"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	defaultShutdownTimeout = 5 * time.Second
	otelName               = "openmeter.io/notification-service"
)

func main() {
	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)
	ctx := context.Background()

	config.SetViperDefaults(v, flags)

	flags.String("config", "", "Configuration file")
	flags.Bool("version", false, "Show version information")

	_ = flags.Parse(os.Args[1:])

	if v, _ := flags.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", "Open Meter", version, revision, revisionDate)

		os.Exit(0)
	}

	if c, _ := flags.GetString("config"); c != "" {
		v.SetConfigFile(c)
	}

	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		panic(err)
	}

	var conf config.Configuration
	err = v.Unmarshal(&conf)
	if err != nil {
		panic(err)
	}

	err = conf.Validate()
	if err != nil {
		panic(err)
	}

	extraResources, _ := resource.New(
		context.Background(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("openmeter"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(conf.Environment),
		),
	)
	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	logger := slog.New(slogmulti.Pipe(
		otelslog.NewHandler,
		contextx.NewLogHandler,
		operation.NewLogHandler,
	).Handler(conf.Telemetry.Log.NewHandler(os.Stdout)))
	logger = otelslog.WithResource(logger, res)

	slog.SetDefault(logger)

	telemetryRouter := chi.NewRouter()
	telemetryRouter.Mount("/debug", middleware.Profiler())

	// Initialize OTel Metrics
	otelMeterProvider, err := conf.Telemetry.Metrics.NewMeterProvider(ctx, res)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry Metrics provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := otelMeterProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down meter provider", slog.String("error", err.Error()))
		}
	}()
	otel.SetMeterProvider(otelMeterProvider)

	metricMeter := otelMeterProvider.Meter(otelName)

	if conf.Telemetry.Metrics.Exporters.Prometheus.Enabled {
		telemetryRouter.Handle("/metrics", promhttp.Handler())
	}

	// Initialize OTel Tracer
	otelTracerProvider, err := conf.Telemetry.Trace.NewTracerProvider(ctx, res)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry Trace provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := otelTracerProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down tracer provider", slog.String("error", err.Error()))
		}
	}()

	otel.SetTracerProvider(otelTracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Validate service prerequisites

	if !conf.Events.Enabled {
		logger.Error("events are disabled, exiting")
		os.Exit(1)
	}

	// Configure health checker
	healthChecker := health.New(health.WithCheckListeners(gosundheit.NewLogger(logger.With(slog.String("component", "healthcheck")))))
	{
		handler := healthhttp.HandleHealthJSON(healthChecker)
		telemetryRouter.Handle("/healthz", handler)

		// Kubernetes style health checks
		telemetryRouter.HandleFunc("/healthz/live", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		telemetryRouter.Handle("/healthz/ready", handler)
	}

	var group run.Group

	// Initialize the data sources (entitlements, productcatalog, etc.)
	// Dependencies: meters
	meterRepository := meter.NewInMemoryRepository(slicesx.Map(conf.Meters, func(meter *models.Meter) models.Meter {
		return *meter
	}))

	// Dependencies: clickhouse
	clickHouseClient, err := clickhouse.Open(conf.Aggregation.ClickHouse.GetClientOptions())
	if err != nil {
		logger.Error("failed to initialize clickhouse client", "error", err)
		os.Exit(1)
	}

	// Dependencies: streamingConnector
	clickhouseStreamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		Logger:               logger,
		ClickHouse:           clickHouseClient,
		Database:             conf.Aggregation.ClickHouse.Database,
		Meters:               meterRepository,
		CreateOrReplaceMeter: conf.Aggregation.CreateOrReplaceMeter,
		PopulateMeter:        conf.Aggregation.PopulateMeter,
	})
	if err != nil {
		logger.Error("failed to initialize clickhouse aggregation", "error", err)
		os.Exit(1)
	}

	// Dependencies: postgresql

	// Initialize Postgres driver
	postgresDriver, err := pgdriver.NewPostgresDriver(
		ctx,
		conf.Postgres.URL,
		pgdriver.WithTracerProvider(otelTracerProvider),
		pgdriver.WithMeterProvider(otelMeterProvider),
	)
	if err != nil {
		logger.Error("failed to initialize postgres driver", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err = postgresDriver.Close(); err != nil {
			logger.Error("failed to close postgres driver", "error", err)
		}
	}()

	// Initialize Ent driver
	entPostgresDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())
	defer func() {
		if err = entPostgresDriver.Close(); err != nil {
			logger.Error("failed to close ent driver", "error", err)
		}
	}()

	entClient := entPostgresDriver.Client()

	// Run database schema creation
	err = entClient.Schema.Create(ctx)
	if err != nil {
		logger.Error("failed to create database schema", "error", err)
		os.Exit(1)
	}

	logger.Info("Postgres client initialized")

	// Create  subscriber
	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            wmBrokerConfiguration(conf, logger, metricMeter),
		ConsumerGroupName: conf.Notification.Consumer.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create publisher
	eventPublisherDriver, err := initEventPublisherDriver(ctx, logger, conf, metricMeter)
	if err != nil {
		logger.Error("failed to initialize event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		// We are using sync producer, so it is fine to close this as a last step
		if err := eventPublisherDriver.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
		}
	}()

	eventPublisher, err := eventbus.New(eventbus.Options{
		Publisher:              eventPublisherDriver,
		Config:                 conf.Events,
		Logger:                 logger,
		MarshalerTransformFunc: kafka.AddPartitionKeyFromSubject,
	})
	if err != nil {
		logger.Error("failed to initialize event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Dependencies: entitlement
	entitlementConnRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     entClient,
		StreamingConnector: clickhouseStreamingConnector,
		MeterRepository:    meterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})

	// Dependencies: notification
	notificationRepo, err := notificationrepository.New(notificationrepository.Config{
		Client: entClient,
		Logger: logger.WithGroup("notification.postgres"),
	})
	if err != nil {
		logger.Error("failed to initialize notification repository", "error", err)
		os.Exit(1)
	}

	notificationWebhook, err := notificationwebhook.New(notificationwebhook.Config{
		SvixConfig: conf.Svix,
	})
	if err != nil {
		logger.Error("failed to initialize notification repository", "error", err)
		os.Exit(1)
	}

	notificationService, err := notification.New(notification.Config{
		Repository:       notificationRepo,
		Webhook:          notificationWebhook,
		FeatureConnector: entitlementConnRegistry.Feature,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		logger.Error("failed to initialize notification service", "error", err)
		os.Exit(1)
	}

	// Initialize consumer
	consumerOptions := consumer.Options{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   eventPublisherDriver,
			Logger:      logger,
			MetricMeter: metricMeter,

			Config: conf.Notification.Consumer,
		},
		Marshaler: eventPublisher.Marshaler(),

		Notification: notificationService,

		Logger: logger,
	}

	notifictionConsumer, err := consumer.New(consumerOptions)
	if err != nil {
		logger.Error("failed to initialize worker", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Run worker components

	// Telemetry server
	server := &http.Server{
		Addr:    conf.Telemetry.Address,
		Handler: telemetryRouter,
	}
	defer server.Close()

	group.Add(
		func() error { return server.ListenAndServe() },
		func(err error) { _ = server.Shutdown(ctx) },
	)

	// Notification service consumer
	group.Add(
		func() error { return notifictionConsumer.Run(ctx) },
		func(err error) { _ = notifictionConsumer.Close() },
	)

	// Handle shutdown
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	// Run the group
	err = group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		slog.Info("received signal; shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

func wmBrokerConfiguration(conf config.Configuration, logger *slog.Logger, metricMeter metric.Meter) kafka.BrokerOptions {
	return kafka.BrokerOptions{
		KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
		ClientID:     otelName,
		Logger:       logger,
		MetricMeter:  metricMeter,
		DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
	}
}

func initEventPublisherDriver(ctx context.Context, logger *slog.Logger, conf config.Configuration, metricMeter metric.Meter) (message.Publisher, error) {
	provisionTopics := []watermillkafka.AutoProvisionTopic{}
	if conf.Notification.Consumer.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, watermillkafka.AutoProvisionTopic{
			Topic:         conf.Notification.Consumer.DLQ.Topic,
			NumPartitions: int32(conf.Notification.Consumer.DLQ.AutoProvision.Partitions),
			Retention:     conf.BalanceWorker.DLQ.AutoProvision.Retention,
		})
	}

	return watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker:          wmBrokerConfiguration(conf, logger, metricMeter),
		ProvisionTopics: provisionTopics,
	})
}
