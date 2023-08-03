package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thmeitz/ksqldb-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/internal/ingest/httpingest"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/ingest/memorydedupe"
	"github.com/openmeterio/openmeter/internal/ingest/redisdedupe"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/server"
	"github.com/openmeterio/openmeter/internal/server/router"
	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
	"github.com/openmeterio/openmeter/internal/streaming/ksqldb_connector"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	"github.com/openmeterio/openmeter/pkg/gosundheit/ksqldbcheck"
)

func main() {
	v, flags := viper.New(), pflag.NewFlagSet("Open Meter", pflag.ExitOnError)
	ctx := context.Background()

	configure(v, flags)

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

	var config configuration
	err = v.Unmarshal(&config)
	if err != nil {
		panic(err)
	}

	err = config.Validate()
	if err != nil {
		panic(err)
	}

	var logger *slog.Logger
	var slogLevel slog.Level

	err = slogLevel.UnmarshalText([]byte(config.Log.Level))
	if err != nil {
		slogLevel = slog.LevelInfo
	}

	switch config.Log.Format {
	case "json":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))

	case "text":
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))

	case "tint":
		logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug}))

	default:
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
	}

	slog.SetDefault(logger)

	telemetryRouter := chi.NewRouter()
	telemetryRouter.Mount("/debug", middleware.Profiler())

	extraResources, _ := resource.New(
		ctx,
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("openmeter"),
		),
	)
	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	exporter, err := prometheus.New()
	if err != nil {
		logger.Error("initializing prometheus exporter: %v", err)
		os.Exit(1)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)
	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down meter provider: %v", err)
		}
	}()

	otel.SetMeterProvider(meterProvider)

	telemetryRouter.Handle("/metrics", promhttp.Handler())

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
		"address":              config.Address,
		"telemetry.address":    config.Telemetry.Address,
		"ingest.kafka.broker":  config.Ingest.Kafka.Broker,
		"processor.ksqldb.url": config.Processor.KSQLDB.URL,
		"schemaRegistry.url":   config.SchemaRegistry.URL,
	})

	var group run.Group
	var ingestCollector ingest.Collector
	var streamingConnector streaming.Connector
	var namespaceHandlers []namespace.Handler

	// Initialize serializer
	eventSerializer, err := initSerializer(config)
	if err != nil {
		logger.Error("failed to initialize serializer", "error", err)
		os.Exit(1)
	}

	// Initialize Kafka Ingest
	ingestCollector, kafkaIngestNamespaceHandler, err := initKafkaIngest(ctx, config, logger, eventSerializer, group)
	if err != nil {
		logger.Error("failed to initialize kafka ingest", "error", err)
		os.Exit(1)
	}
	namespaceHandlers = append(namespaceHandlers, kafkaIngestNamespaceHandler)
	defer ingestCollector.Close()

	// Initialize ksqlDB Streaming Processor
	if config.Processor.KSQLDB.Enabled {
		ksqlDBStreamingConnector, ksqlDBNamespaceHandler, err := initKsqlDBStreaming(config, logger, eventSerializer, healthChecker)
		if err != nil {
			logger.Error("failed to initialize ksqldb streaming processor", "error", err)
			os.Exit(1)
		}
		streamingConnector = ksqlDBStreamingConnector
		namespaceHandlers = append(namespaceHandlers, ksqlDBNamespaceHandler)
	}

	// Initialize ClickHouse Streaming Processor
	if config.Processor.ClickHouse.Enabled {
		clickhouseStreamingConnector, err := initClickHouseStreaming(config, logger)
		if err != nil {
			logger.Error("failed to initialize clickhouse streaming processor", "error", err)
			os.Exit(1)
		}
		streamingConnector = clickhouseStreamingConnector
		namespaceHandlers = append(namespaceHandlers, clickhouseStreamingConnector)
	}

	// Initialize Namespace
	namespaceManager, err := initNamespace(config, namespaceHandlers...)
	if err != nil {
		logger.Error("failed to initialize namespace", "error", err)
		os.Exit(1)
	}

	// Initialize Memory Dedupe
	if config.Dedupe.Memory.Enabled {
		ingestCollector, err = memorydedupe.NewCollector(memorydedupe.CollectorConfig{
			Collector: ingestCollector,
			Size:      config.Dedupe.Memory.Size,
		})
		if err != nil {
			logger.Error("failed to initialize memory dedupe", "error", err)
			os.Exit(1)
		}
	}

	// Initialize Redis Dedupe
	if config.Dedupe.Redis.Enabled {
		ingestCollector, err = initDedupeRedis(config, logger, ingestCollector)
		if err != nil {
			logger.Error("failed to initialize redis dedupe", "error", err)
			os.Exit(1)
		}
	}

	// Initialize HTTP Ingest handler
	ingestHandler, err := httpingest.NewHandler(httpingest.HandlerConfig{
		Collector:        ingestCollector,
		NamespaceManager: namespaceManager,
		Logger:           logger,
	})
	if err != nil {
		logger.Error("failed to initialize http ingest handler", "error", err)
		os.Exit(1)
	}

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			NamespaceManager:   namespaceManager,
			StreamingConnector: streamingConnector,
			IngestHandler:      ingestHandler,
			Meters:             config.Meters,
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
					otelhttp.WithMeterProvider(meterProvider),
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

	for _, meter := range config.Meters {
		err := streamingConnector.CreateMeter(ctx, namespaceManager.GetDefaultNamespace(), meter)
		if err != nil {
			slog.Warn("failed to initialize meter", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("meters successfully created", "count", len(config.Meters))

	// Set up telemetry server
	{
		server := &http.Server{
			Addr:    config.Telemetry.Address,
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
			Addr:    config.Address,
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

func initKafkaIngest(ctx context.Context, config configuration, logger *slog.Logger, serializer serializer.Serializer, group run.Group) (*kafkaingest.Collector, *kafkaingest.NamespaceHandler, error) {
	// Initialize Kafka Admin Client
	kafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	kafkaAdminClient, err := kafka.NewAdminClient(kafkaConfig)
	if err != nil {
		return nil, nil, err
	}

	namespaceHandler := &kafkaingest.NamespaceHandler{
		AdminClient:             kafkaAdminClient,
		NamespacedTopicTemplate: config.Ingest.Kafka.EventsTopicTemplate,
		Partitions:              config.Ingest.Kafka.Partitions,
		Logger:                  logger,
	}

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, namespaceHandler, fmt.Errorf("init kafka ingest: %w", err)
	}
	group.Add(kafkaingest.KafkaProducerGroup(ctx, producer, logger))

	slog.Debug("connected to Kafka")

	collector := &kafkaingest.Collector{
		Producer:                producer,
		NamespacedTopicTemplate: config.Ingest.Kafka.EventsTopicTemplate,
		Serializer:              serializer,
	}

	return collector, namespaceHandler, nil
}

// initSerializer initializes the serializer based on the configuration.
func initSerializer(config configuration) (serializer.Serializer, error) {
	// Initialize JSON_SR with Schema Registry
	if config.SchemaRegistry.URL != "" {
		schemaRegistryConfig := schemaregistry.NewConfig(config.SchemaRegistry.URL)
		if config.SchemaRegistry.Username != "" || config.SchemaRegistry.Password != "" {
			schemaRegistryConfig.BasicAuthCredentialsSource = "USER_INFO"
			schemaRegistryConfig.BasicAuthUserInfo = fmt.Sprintf("%s:%s", config.SchemaRegistry.Username, config.SchemaRegistry.Password)
		}
		schemaRegistry, err := schemaregistry.NewClient(schemaRegistryConfig)
		if err != nil {
			return nil, fmt.Errorf("init serializer: %w", err)
		}

		return serializer.NewJSONSchemaSerializer(schemaRegistry)
	} else {
		// Initialize JSON without Schema Registry
		return serializer.NewJSONSerializer(), nil
	}
}

func initKsqlDBStreaming(config configuration, logger *slog.Logger, serializer serializer.Serializer, healthChecker health.Health) (*ksqldb_connector.KsqlDBConnector, *ksqldb_connector.NamespaceHandler, error) {
	// Initialize ksqlDB Client
	ksqldbClient, err := ksqldb.NewClientWithOptions(config.Processor.KSQLDB.CreateKSQLDBConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("init ksqldb streaming: %w", err)
	}
	defer ksqldbClient.Close()

	// Register KSQLDB health check
	err = healthChecker.RegisterCheck(
		ksqldbcheck.NewCheck("ksqldb", ksqldbClient),
		health.ExecutionPeriod(5*time.Second),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("init ksqldb streaming: %w", err)
	}

	namespaceHandler := &ksqldb_connector.NamespaceHandler{
		KsqlDBClient:                          &ksqldbClient,
		NamespacedEventsTopicTemplate:         config.Ingest.Kafka.EventsTopicTemplate,
		NamespacedDetectedEventsTopicTemplate: config.Processor.KSQLDB.DetectedEventsTopicTemplate,
		Format:                                serializer.GetFormat(),
		KeySchemaID:                           serializer.GetKeySchemaId(),
		ValueSchemaID:                         serializer.GetValueSchemaId(),
		Partitions:                            config.Ingest.Kafka.Partitions,
	}

	connector, err := ksqldb_connector.NewKsqlDBConnector(&ksqldbClient, config.Ingest.Kafka.Partitions, serializer.GetFormat(), logger)
	if err != nil {
		return nil, nil, fmt.Errorf("init ksqldb streaming: %w", err)
	}

	return connector, namespaceHandler, nil
}

func initClickHouseStreaming(config configuration, logger *slog.Logger) (*clickhouse_connector.ClickhouseConnector, error) {
	options := &clickhouse.Options{
		Addr: []string{config.Processor.ClickHouse.Address},
		Auth: clickhouse.Auth{
			Database: config.Processor.ClickHouse.Database,
			Username: config.Processor.ClickHouse.Username,
			Password: config.Processor.ClickHouse.Password,
		},
		DialTimeout:      time.Duration(10) * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Duration(10) * time.Minute,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  10,
	}
	// This minimal TLS.Config is normally sufficient to connect to the secure native port (normally 9440) on a ClickHouse server.
	// See: https://clickhouse.com/docs/en/integrations/go#using-tls
	if config.Processor.ClickHouse.TLS {
		options.TLS = &tls.Config{}
	}

	// Initialize ClickHouse
	clickHouseClient, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("init clickhouse client: %w", err)
	}

	kafkaConnect, err := sink.NewKafkaConnect(sink.KafkaConnectConfig{
		Logger: logger,
		URL:    config.Sink.KafkaConnect.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("init kafka connect: %w", err)
	}

	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		Logger:              logger,
		KafkaConnect:        kafkaConnect,
		KafkaConnectEnabled: config.Sink.KafkaConnect.Enabled,
		SinkConfig: clickhouse_connector.SinkConfig{
			Hostname: config.Sink.KafkaConnect.ClickHouse.Hostname,
			Port:     config.Sink.KafkaConnect.ClickHouse.Port,
			SSL:      config.Sink.KafkaConnect.ClickHouse.SSL,
			Username: config.Sink.KafkaConnect.ClickHouse.Username,
			Password: config.Sink.KafkaConnect.ClickHouse.Password,
			Database: config.Sink.KafkaConnect.ClickHouse.Database,
		},
		ClickHouse: clickHouseClient,
		Database:   config.Processor.ClickHouse.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	return streamingConnector, nil
}

// initDedupe initializes the dedupe based on the configuration.
func initDedupeRedis(config configuration, logger *slog.Logger, collector ingest.Collector) (*redisdedupe.Collector, error) {
	// Initialize Redis
	var redisClient *redis.Client

	if config.Dedupe.Redis.UseSentinel {
		redisClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    config.Dedupe.Redis.MasterName,
			SentinelAddrs: []string{config.Dedupe.Redis.Address},
			// RouteByLatency:          false,
			// RouteRandomly:           false,
			Password: config.Dedupe.Redis.Password,
			Username: config.Dedupe.Redis.Username,
			DB:       config.Dedupe.Redis.Database,
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.Dedupe.Redis.Address,
			Password: config.Dedupe.Redis.Password,
			Username: config.Dedupe.Redis.Username,
			DB:       config.Dedupe.Redis.Database,
		})
	}

	dedupeRedis, err := redisdedupe.NewCollector(redisdedupe.CollectorConfig{
		Logger:     logger,
		Redis:      redisClient,
		Expiration: config.Dedupe.Redis.Expiration,
		Collector:  collector,
	})
	if err != nil {
		return nil, fmt.Errorf("init redis dedupe: %w", err)
	}

	return dedupeRedis, nil
}

func initNamespace(config configuration, namespaces ...namespace.Handler) (*namespace.Manager, error) {
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
