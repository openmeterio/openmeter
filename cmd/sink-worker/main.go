package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sagikazarmark/slog-shim"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var otelName string = "openmeter.io/sink-worker"

func main() {
	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)
	ctx := context.Background()

	config.Configure(v, flags)

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
	err = v.Unmarshal(&conf, viper.DecodeHook(config.DecodeHook()))
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
			semconv.ServiceName("openmeter-sink-worker"),
			semconv.ServiceVersion(version),
			attribute.String("environment", conf.Environment),
		),
	)
	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	logger := slog.New(otelslog.NewHandler(conf.Telemetry.Log.NewHandler(os.Stdout)))
	logger = otelslog.WithResource(logger, res)

	slog.SetDefault(logger)

	telemetryRouter := chi.NewRouter()
	telemetryRouter.Mount("/debug", middleware.Profiler())

	// Initialize OTel Metrics
	otelMeterProvider, err := conf.Telemetry.Metrics.NewMeterProvider(context.Background(), res)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := otelMeterProvider.Shutdown(context.Background()); err != nil {
			logger.Error("shutting down meter provider: %v", err)
		}
	}()
	otel.SetMeterProvider(otelMeterProvider)
	metricMeter := otelMeterProvider.Meter(otelName)

	if conf.Telemetry.Metrics.Exporters.Prometheus.Enabled {
		telemetryRouter.Handle("/metrics", promhttp.Handler())
	}

	// Initialize OTel Tracer
	otelTracerProvider, err := conf.Telemetry.Trace.NewTracerProvider(context.Background(), res)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := otelTracerProvider.Shutdown(context.Background()); err != nil {
			logger.Error("shutting down tracer provider", "error", err)
		}
	}()

	otel.SetTracerProvider(otelTracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracer := otelTracerProvider.Tracer(otelName)

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

	logger.Info("starting OpenMeter sink worker", "config", map[string]string{
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	// Initialize meter repository
	meterRepository := meter.NewInMemoryRepository(slicesx.Map(conf.Meters, func(meter *models.Meter) models.Meter {
		return *meter
	}))

	// Initialize sink worker
	sink, err := initSink(conf, logger, metricMeter, tracer, meterRepository)
	if err != nil {
		logger.Error("failed to initialize sink worker", "error", err)
		os.Exit(1)
	}

	var group run.Group

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

	// Starting sink worker
	{
		defer sink.Close()

		group.Add(
			func() error { return sink.Run() },
			func(err error) { _ = sink.Close() },
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

func initClickHouseClient(config config.Configuration) (clickhouse.Conn, error) {
	options := &clickhouse.Options{
		Addr: []string{config.Aggregation.ClickHouse.Address},
		Auth: clickhouse.Auth{
			Database: config.Aggregation.ClickHouse.Database,
			Username: config.Aggregation.ClickHouse.Username,
			Password: config.Aggregation.ClickHouse.Password,
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
	if config.Aggregation.ClickHouse.TLS {
		options.TLS = &tls.Config{}
	}

	// Initialize ClickHouse
	clickHouseClient, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("init clickhouse client: %w", err)
	}

	return clickHouseClient, nil
}

func initSink(config config.Configuration, logger *slog.Logger, metricMeter metric.Meter, tracer trace.Tracer, meterRepository meter.Repository) (*sink.Sink, error) {
	clickhouseClient, err := initClickHouseClient(config)
	if err != nil {
		return nil, fmt.Errorf("init clickhouse client: %w", err)
	}

	var deduplicator dedupe.Deduplicator
	if config.Sink.Dedupe.Enabled {
		deduplicator, err = config.Sink.Dedupe.NewDeduplicator()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize deduplicator: %w", err)
		}
	}

	storage := sink.NewClickhouseStorage(
		sink.ClickHouseStorageConfig{
			ClickHouse: clickhouseClient,
			Database:   config.Aggregation.ClickHouse.Database,
		},
	)

	kafkaAdminConfig := config.Ingest.Kafka.CreateKafkaConfig()
	kafkaAdminClient, err := kafka.NewAdminClient(&kafkaAdminConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka admin client: %w", err)
	}

	consumerKafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	_ = consumerKafkaConfig.SetKey("group.id", config.Sink.GroupId)

	producerKafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()

	sinkConfig := sink.SinkConfig{
		Logger:              logger,
		Tracer:              tracer,
		MetricMeter:         metricMeter,
		MeterRepository:     meterRepository,
		Storage:             storage,
		Deduplicator:        deduplicator,
		KafkaAdminClient:    kafkaAdminClient,
		ConsumerKafkaConfig: consumerKafkaConfig,
		ProducerKafkaConfig: producerKafkaConfig,
		MinCommitCount:      config.Sink.MinCommitCount,
		MaxCommitWait:       config.Sink.MaxCommitWait,
		NamespaceRefetch:    config.Sink.NamespaceRefetch,
	}

	return sink.NewSink(sinkConfig)
}
