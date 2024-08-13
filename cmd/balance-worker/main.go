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

	"entgo.io/ent/dialect/sql"
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
	"github.com/openmeterio/openmeter/internal/ent/db"
	entitlementpgadapter "github.com/openmeterio/openmeter/internal/entitlement/adapter"
	"github.com/openmeterio/openmeter/internal/entitlement/balanceworker"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/registry"
	registrybuilder "github.com/openmeterio/openmeter/internal/registry/builder"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
	"github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
	watermillkafka "github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/internal/watermill/router"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	defaultShutdownTimeout = 5 * time.Second
	otelName               = "openmeter.io/balance-worker"
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
	pgClients, err := initPGClients(conf.Postgres)
	if err != nil {
		logger.Error("failed to initialize postgres clients", "error", err)
		os.Exit(1)
	}
	defer pgClients.driver.Close()

	logger.Info("Postgres clients initialized")

	// Create  subscriber
	wmBrokerConfig := wmBrokerConfiguration(conf, logger, metricMeter)

	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            wmBrokerConfig,
		ConsumerGroupName: conf.BalanceWorker.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create publisher
	eventPublisherDriver, err := initEventPublisherDriver(ctx, wmBrokerConfig, conf)
	if err != nil {
		logger.Error("failed to initialize event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		// We are using sync publishing, so it's fine to close the publisher using defers.
		if err := eventPublisherDriver.Close(); err != nil {
			logger.Error("failed to close event publisher", slog.String("error", err.Error()))
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
	entitlementConnectors := registrybuilder.GetEntitlementRegistry(registry.EntitlementOptions{
		DatabaseClient:     pgClients.client,
		StreamingConnector: clickhouseStreamingConnector,
		MeterRepository:    meterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})

	// Initialize worker
	workerOptions := balanceworker.WorkerOptions{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		IngestEventsTopic: conf.Events.IngestEvents.Topic,

		Router: router.Options{
			Subscriber: wmSubscriber,
			Publisher:  eventPublisherDriver,
			Logger:     logger,

			Config: conf.BalanceWorker.ConsumerConfiguration,
		},

		EventBus: eventPublisher,

		Entitlement: entitlementConnectors,
		Repo:        entitlementpgadapter.NewPostgresEntitlementRepo(pgClients.client),

		Logger: logger,
	}

	worker, err := balanceworker.New(workerOptions)
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

	// Balance worker
	group.Add(
		func() error { return worker.Run(ctx) },
		func(err error) { _ = worker.Close() },
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

func wmBrokerConfiguration(conf config.Configuration, logger *slog.Logger, metricMeter metric.Meter) watermillkafka.BrokerOptions {
	return watermillkafka.BrokerOptions{
		KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
		ClientID:     otelName,
		Logger:       logger,
		MetricMeter:  metricMeter,
		DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
	}
}

func initEventPublisherDriver(ctx context.Context, broker watermillkafka.BrokerOptions, conf config.Configuration) (message.Publisher, error) {
	provisionTopics := []watermillkafka.AutoProvisionTopic{}
	if conf.BalanceWorker.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, watermillkafka.AutoProvisionTopic{
			Topic:         conf.BalanceWorker.DLQ.Topic,
			NumPartitions: int32(conf.BalanceWorker.DLQ.AutoProvision.Partitions),
		})
	}

	return watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker:          broker,
		ProvisionTopics: provisionTopics,
	})
}

type pgClients struct {
	driver *sql.Driver
	client *db.Client
}

func initPGClients(config config.PostgresConfig) (
	*pgClients,
	error,
) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}
	driver, err := entutils.GetPGDriver(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres driver: %w", err)
	}

	return &pgClients{
		driver: driver,
		client: db.NewClient(db.Driver(driver)),
	}, nil
}
