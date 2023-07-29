package main

import (
	"context"
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
		context.Background(),
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
		if err := meterProvider.Shutdown(context.Background()); err != nil {
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
	namespaceHandlers := make([]namespace.Handler, 0)

	// Initialize serializer
	eventSerializer, err := initSerializer(config)
	if err != nil {
		slog.Error("failed to initialize serializer", "error", err)
		os.Exit(1)
	}

	// Initialize Kafka Ingest
	ingestCollector, kafkaIngestNamespaceHandler, err := initKafkaIngest(config, logger, eventSerializer, group)
	if err != nil {
		slog.Error("failed to initialize kafka ingest", "error", err)
		os.Exit(1)
	}
	namespaceHandlers = append(namespaceHandlers, kafkaIngestNamespaceHandler)
	defer ingestCollector.Close()

	// Initialize ksqlDB Streaming Processor
	if config.Processor.KSQLDB.Enabled {
		ksqlDBStreamingConnector, ksqlDBNamespaceHandler, err := initKsqlDBStreaming(config, logger, eventSerializer, healthChecker)
		if err != nil {
			slog.Error("failed to initialize ksqldb streaming processor", "error", err)
			os.Exit(1)
		}
		streamingConnector = ksqlDBStreamingConnector
		namespaceHandlers = append(namespaceHandlers, ksqlDBNamespaceHandler)
	}

	// Initialize ClickHouse Streaming Processor
	if config.Processor.ClickHouse.Enabled {
		clickhouseStreamingConnector, err := initClickHouseStreaming(config, logger)
		if err != nil {
			slog.Error("failed to initialize clickhouse streaming processor", "error", err)
			os.Exit(1)
		}
		streamingConnector = clickhouseStreamingConnector
		namespaceHandlers = append(namespaceHandlers, clickhouseStreamingConnector)
	}

	// Initialize Namespace
	namespaceManager, err := initNamespace(namespaceHandlers...)
	if err != nil {
		slog.Error("failed to initialize namespace", "error", err)
		os.Exit(1)
	}

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			NamespaceManager:   namespaceManager,
			StreamingConnector: streamingConnector,
			IngestHandler: httpingest.Handler{
				Collector: ingestCollector,
				Logger:    logger,
			},
			Meters: config.Meters,
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
		slog.Error("failed to create server", "error", err)
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
		err := streamingConnector.Init(meter, namespace.DefaultNamespace)
		if err != nil {
			slog.Warn("failed to initialize meter", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("meters successfully initialized", "count", len(config.Meters))

	// Set up telemetry server
	{
		server := &http.Server{
			Addr:    config.Telemetry.Address,
			Handler: telemetryRouter,
		}
		defer server.Close()

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(context.Background()) },
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
			func(err error) { _ = server.Shutdown(context.Background()) }, // TODO: context deadline
		)
	}

	// Setup signal handler
	group.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		slog.Info("received signal; shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		slog.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

func initKafkaIngest(config configuration, logger *slog.Logger, serializer serializer.Serializer, group run.Group) (*kafkaingest.Collector, *kafkaingest.NamespaceHandler, error) {
	// Initialize Kafka Admin Client
	kafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	kafkaAdminClient, err := kafka.NewAdminClient(kafkaConfig)
	if err != nil {
		return nil, nil, err
	}

	namespaceHandler := &kafkaingest.NamespaceHandler{
		AdminClient:             kafkaAdminClient,
		NamespacedTopicTemplate: config.Namespace.EventsTopicTemplate,
		Partitions:              config.Ingest.Kafka.Partitions,
		Logger:                  logger,
	}

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		return nil, namespaceHandler, fmt.Errorf("init kafka ingest: %w", err)
	}
	group.Add(kafkaingest.KafkaProducerGroup(context.Background(), producer, logger))

	slog.Debug("connected to Kafka")

	collector := &kafkaingest.Collector{
		Producer:                producer,
		NamespacedTopicTemplate: config.Namespace.EventsTopicTemplate,
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
		NamespacedEventsTopicTemplate:         config.Namespace.EventsTopicTemplate,
		NamespacedDetectedEventsTopicTemplate: config.Namespace.DetectedEventsTopicTemplate,
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
	// Initialize ClickHouse
	clickHouseClient, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{config.Processor.ClickHouse.Address},
		Auth: clickhouse.Auth{
			Database: config.Processor.ClickHouse.Database,
			Username: config.Processor.ClickHouse.Username,
			Password: config.Processor.ClickHouse.Password,
		},
		// TLS: &tls.Config{
		// 	InsecureSkipVerify: true,
		// },
		DialTimeout:      time.Duration(10) * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Duration(10) * time.Minute,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  10,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse client: %w", err)
	}

	kafkaConnect, err := sink.NewKafkaConnect(&sink.KafkaConnectConfig{
		Address: config.Sink.KafkaConnect.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("init kafka connect: %w", err)
	}

	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(&clickhouse_connector.ClickhouseConnectorConfig{
		Logger:       logger,
		KafkaConnect: kafkaConnect,
		ClickHouse:   clickHouseClient,
		Database:     config.Processor.ClickHouse.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	return streamingConnector, nil
}

func initNamespace(namespaces ...namespace.Handler) (namespace.Manager, error) {
	namespaceManager := namespace.Manager{
		Handlers: namespaces,
	}

	slog.Debug("create default namespace")
	err := namespaceManager.CreateDefaultNamespace(context.Background())
	if err != nil {
		return namespaceManager, fmt.Errorf("create default namespace: %v", err)
	}
	slog.Info("default namespace created")
	return namespaceManager, nil
}
