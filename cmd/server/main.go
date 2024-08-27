package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/debug"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/driver"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	defaultShutdownTimeout = 5 * time.Second
	otelName               = "openmeter.io/backend"
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

	logger.Info("starting OpenMeter server", "config", map[string]string{
		"address":             conf.Address,
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	var group run.Group
	var ingestCollector ingest.Collector
	var streamingConnector streaming.Connector
	var namespaceHandlers []namespace.Handler

	// Initialize ClickHouse Client
	clickHouseClient, err := clickhouse.Open(conf.Aggregation.ClickHouse.GetClientOptions())
	if err != nil {
		logger.Error("failed to initialize clickhouse client", "error", err)
		os.Exit(1)
	}

	kafkaProducer, err := initKafkaProducer(ctx, conf, logger, metricMeter, &group)
	if err != nil {
		logger.Error("failed to initialize kafka producer", "error", err)
		os.Exit(1)
	}

	eventPublisherDriver, err := initEventPublisherDriver(ctx, logger, conf, metricMeter)
	if err != nil {
		logger.Error("failed to initialize event publisher", "error", err)
		os.Exit(1)
	}

	defer func() {
		logger.Info("closing event publisher")
		if err = eventPublisherDriver.Close(); err != nil {
			logger.Error("failed to close event publisher", "error", err)
		}
	}()

	eventPublisher, err := eventbus.New(eventbus.Options{
		Publisher:              eventPublisherDriver,
		Config:                 conf.Events,
		Logger:                 logger,
		MarshalerTransformFunc: watermillkafka.AddPartitionKeyFromSubject,
	})
	if err != nil {
		logger.Error("failed to initialize event bus", "error", err)
		os.Exit(1)
	}

	// Initialize Kafka Ingest
	ingestCollector, kafkaIngestNamespaceHandler, err := initKafkaIngest(
		kafkaProducer,
		conf,
		logger,
		metricMeter,
		serializer.NewJSONSerializer(),
	)
	if err != nil {
		logger.Error("failed to initialize kafka ingest", "error", err)
		os.Exit(1)
	}
	namespaceHandlers = append(namespaceHandlers, kafkaIngestNamespaceHandler)
	defer ingestCollector.Close()

	meterRepository := meter.NewInMemoryRepository(slicesx.Map(conf.Meters, func(meter *models.Meter) models.Meter {
		return *meter
	}))

	// Initialize ClickHouse Aggregation
	clickhouseStreamingConnector, err := initClickHouseStreaming(conf, clickHouseClient, meterRepository, logger)
	if err != nil {
		logger.Error("failed to initialize clickhouse aggregation", "error", err)
		os.Exit(1)
	}

	streamingConnector = clickhouseStreamingConnector
	namespaceHandlers = append(namespaceHandlers, clickhouseStreamingConnector)

	// Initialize Namespace
	namespaceManager, err := initNamespace(conf, namespaceHandlers...)
	if err != nil {
		logger.Error("failed to initialize namespace", "error", err)
		os.Exit(1)
	}

	// Initialize deduplication
	if conf.Dedupe.Enabled {
		deduplicator, err := conf.Dedupe.NewDeduplicator()
		if err != nil {
			logger.Error("failed to initialize deduplicator", "error", err)
			os.Exit(1)
		}
		defer func() {
			logger.Info("closing deduplicator")
			if err = deduplicator.Close(); err != nil {
				logger.Error("failed to close deduplicator", "error", err)
			}
		}()

		ingestCollector = ingest.DeduplicatingCollector{
			Collector:    ingestCollector,
			Deduplicator: deduplicator,
		}
	}

	// Initialize HTTP Ingest handler
	ingestService := ingest.Service{
		Collector: ingestCollector,
		Logger:    logger,
	}
	ingestHandler := ingestdriver.NewIngestEventsHandler(
		ingestService.IngestEvents,
		namespacedriver.StaticNamespaceDecoder(namespaceManager.GetDefaultNamespace()),
		nil,
		errorsx.NewContextHandler(errorsx.NewAppHandler(errorsx.NewSlogHandler(logger))),
	)

	// Initialize portal
	var portalTokenStrategy *authenticator.PortalTokenStrategy
	if conf.Portal.Enabled {
		portalTokenStrategy, err = authenticator.NewPortalTokenStrategy(conf.Portal.TokenSecret, conf.Portal.TokenExpiration)
		if err != nil {
			logger.Error("failed to initialize portal token strategy", "error", err)
			os.Exit(1)
		}
	}

	debugConnector := debug.NewDebugConnector(streamingConnector)
	entitlementConnRegistry := &registry.Entitlement{}

	var entClient *entdb.Client
	if conf.Entitlements.Enabled {
		// Initialize Postgres driver
		var postgresDriver *pgdriver.Driver
		postgresDriver, err = pgdriver.NewPostgresDriver(
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

		entClient = entPostgresDriver.Client()

		// Run database schema creation
		err = entClient.Schema.Create(ctx)
		if err != nil {
			logger.Error("failed to create schema in database", "error", err)
			os.Exit(1)
		}

		logger.Info("Postgres client initialized")

		entitlementConnRegistry = registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
			DatabaseClient:     entClient,
			StreamingConnector: streamingConnector,
			MeterRepository:    meterRepository,
			Logger:             logger,
			Publisher:          eventPublisher,
		})
	}

	var notificationService notification.Service
	if conf.Notification.Enabled {
		if !conf.Entitlements.Enabled {
			logger.Error("failed to initialize notification service: entitlements must be enabled")
			os.Exit(1)
		}

		// CreatingPG client is done as part of entitlements initialization
		if entClient == nil {
			logger.Error("failed to initialize notification service: postgres client is not initialized")
			os.Exit(1)
		}

		var notificationRepo notification.Repository
		notificationRepo, err = notificationrepository.New(notificationrepository.Config{
			Client: entClient,
			Logger: logger.WithGroup("notification.postgres"),
		})
		if err != nil {
			logger.Error("failed to initialize notification repository", "error", err)
			os.Exit(1)
		}

		var notificationWebhook notificationwebhook.Handler
		notificationWebhook, err = notificationwebhook.New(notificationwebhook.Config{
			SvixConfig: conf.Svix,
		})
		if err != nil {
			logger.Error("failed to initialize notification webhook handler", "error", err)
			os.Exit(1)
		}

		notificationService, err = notification.New(notification.Config{
			Repository:       notificationRepo,
			Webhook:          notificationWebhook,
			FeatureConnector: entitlementConnRegistry.Feature,
			Logger:           logger.With(slog.String("subsystem", "notification")),
		})
		if err != nil {
			logger.Error("failed to initialize notification service", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err = notificationService.Close(); err != nil {
				logger.Error("failed to close notification service", "error", err)
			}
		}()
	}

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			NamespaceManager:    namespaceManager,
			StreamingConnector:  streamingConnector,
			IngestHandler:       ingestHandler,
			Meters:              meterRepository,
			PortalTokenStrategy: portalTokenStrategy,
			PortalCORSEnabled:   conf.Portal.CORS.Enabled,
			ErrorHandler:        errorsx.NewAppHandler(errorsx.NewSlogHandler(logger)),
			// deps
			DebugConnector:              debugConnector,
			FeatureConnector:            entitlementConnRegistry.Feature,
			EntitlementConnector:        entitlementConnRegistry.Entitlement,
			EntitlementBalanceConnector: entitlementConnRegistry.MeteredEntitlement,
			GrantConnector:              entitlementConnRegistry.Grant,
			GrantRepo:                   entitlementConnRegistry.GrantRepo,
			Notification:                notificationService,
			// modules
			EntitlementsEnabled: conf.Entitlements.Enabled,
			NotificationEnabled: conf.Notification.Enabled,
		},
		RouterHook: func(r chi.Router) {
			r.Use(func(h http.Handler) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						h.ServeHTTP(w, r)

						routePattern := chi.RouteContext(r.Context()).RoutePattern()

						span := trace.SpanFromContext(r.Context())
						span.SetName(routePattern)
						span.SetAttributes(semconv.HTTPTarget(r.URL.String()), semconv.HTTPRoute(routePattern))

						labeler, ok := otelhttp.LabelerFromContext(r.Context())
						if ok {
							labeler.Add(semconv.HTTPRoute(routePattern))
						}
					}),
					"",
					otelhttp.WithMeterProvider(otelMeterProvider),
					otelhttp.WithTracerProvider(otelTracerProvider),
				)
			})
		},
	})
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	s.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version": version,
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
		})
	})

	for _, meter := range conf.Meters {
		err := streamingConnector.CreateMeter(ctx, namespaceManager.GetDefaultNamespace(), meter)
		if err != nil {
			slog.Warn("failed to initialize meter", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("meters successfully created", "count", len(conf.Meters))

	// Set up telemetry server
	{
		server := &http.Server{
			Addr:    conf.Telemetry.Address,
			Handler: telemetryRouter,
		}
		defer server.Close()

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) },
		)
	}

	// Set up server
	{
		server := &http.Server{
			Addr:    conf.Address,
			Handler: s,
		}
		defer server.Close()

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) }, // TODO: context deadline
		)
	}

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		slog.Info("received signal; shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

func initEventPublisherDriver(ctx context.Context, logger *slog.Logger, conf config.Configuration, metricMeter metric.Meter) (message.Publisher, error) {
	if !conf.Events.Enabled {
		return &noop.Publisher{}, nil
	}

	provisionTopics := []watermillkafka.AutoProvisionTopic{}
	if conf.Events.SystemEvents.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, watermillkafka.AutoProvisionTopic{
			Topic:         conf.Events.SystemEvents.Topic,
			NumPartitions: int32(conf.Events.SystemEvents.AutoProvision.Partitions),
		})
	}

	return watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:     otelName,
			Logger:       logger,
			MetricMeter:  metricMeter,
			DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
		},
		ProvisionTopics: provisionTopics,
	})
}

func initKafkaProducer(ctx context.Context, config config.Configuration, logger *slog.Logger, metricMeter metric.Meter, group *run.Group) (*kafka.Producer, error) {
	// Initialize Kafka Admin Client
	kafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(&kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("init kafka ingest: %w", err)
	}

	// Initialize Kafka Client Statistics reporter
	kafkaMetrics, err := kafkametrics.New(metricMeter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client metrics: %w", err)
	}

	// TODO: move kafkaingest.KafkaProducerGroup to pkg/kafka
	group.Add(kafkaingest.KafkaProducerGroup(ctx, producer, logger, kafkaMetrics))

	go pkgkafka.ConsumeLogChannel(producer, logger.WithGroup("kafka").WithGroup("producer"))

	slog.Debug("connected to Kafka")
	return producer, nil
}

func initKafkaIngest(producer *kafka.Producer, config config.Configuration, logger *slog.Logger, metricMeter metric.Meter, serializer serializer.Serializer) (*kafkaingest.Collector, *kafkaingest.NamespaceHandler, error) {
	collector, err := kafkaingest.NewCollector(
		producer,
		serializer,
		config.Ingest.Kafka.EventsTopicTemplate,
		metricMeter,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("init kafka ingest: %w", err)
	}

	kafkaAdminClient, err := kafka.NewAdminClientFromProducer(producer)
	if err != nil {
		return nil, nil, err
	}

	namespaceHandler := &kafkaingest.NamespaceHandler{
		AdminClient:             kafkaAdminClient,
		NamespacedTopicTemplate: config.Ingest.Kafka.EventsTopicTemplate,
		Partitions:              config.Ingest.Kafka.Partitions,
		Logger:                  logger,
	}

	return collector, namespaceHandler, nil
}

func initClickHouseStreaming(config config.Configuration, clickHouseClient clickhouse.Conn, meterRepository meter.Repository, logger *slog.Logger) (*clickhouse_connector.ClickhouseConnector, error) {
	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		Logger:               logger,
		ClickHouse:           clickHouseClient,
		Database:             config.Aggregation.ClickHouse.Database,
		Meters:               meterRepository,
		CreateOrReplaceMeter: config.Aggregation.CreateOrReplaceMeter,
		PopulateMeter:        config.Aggregation.PopulateMeter,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	return streamingConnector, nil
}

func initNamespace(config config.Configuration, namespaces ...namespace.Handler) (*namespace.Manager, error) {
	namespaceManager, err := namespace.NewManager(namespace.ManagerConfig{
		Handlers:          namespaces,
		DefaultNamespace:  config.Namespace.Default,
		DisableManagement: config.Namespace.DisableManagement,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace manager: %v", err)
	}

	slog.Debug("create default namespace")
	err = namespaceManager.CreateDefaultNamespace(context.Background())
	if err != nil {
		return nil, fmt.Errorf("create default namespace: %v", err)
	}
	slog.Info("default namespace created")
	return namespaceManager, nil
}
