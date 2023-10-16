package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sagikazarmark/slog-shim"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
)

func main() {
	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

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
			semconv.ServiceName("openmeter"),
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

	meterProvider, err := conf.Telemetry.Metrics.NewMeterProvider(context.Background(), res)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			logger.Error("shutting down meter provider: %v", err)
		}
	}()

	otel.SetMeterProvider(meterProvider)

	if conf.Telemetry.Metrics.Exporters.Prometheus.Enabled {
		telemetryRouter.Handle("/metrics", promhttp.Handler())
	}

	tracerProvider, err := conf.Telemetry.Trace.NewTracerProvider(context.Background(), res)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			logger.Error("shutting down tracer provider", "error", err)
		}
	}()

	otel.SetTracerProvider(tracerProvider)
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

	logger.Info("starting OpenMeter sink worker", "config", map[string]string{
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	// Initialize sink worker
	sink, err := initSink(conf, logger)
	if err != nil {
		logger.Error("failed to initialize sink worker", "error", err)
		os.Exit(1)
	}

	// Starting sink worker
	{
		logger.Info("sink worker started")
		err = sink.Run()
		if err != nil {
			slog.Error("sink error", "error", err)
			os.Exit(1)
		}
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

func initSink(config config.Configuration, logger *slog.Logger) (*sink.Sink, error) {
	clickhouseClient, err := initClickHouseClient(config)
	if err != nil {
		return nil, fmt.Errorf("init clickhouse client: %w", err)
	}

	var deduplicator dedupe.Deduplicator
	if config.Dedupe.Enabled {
		deduplicator, err = config.Dedupe.NewDeduplicator()
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

	consumerKafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	_ = consumerKafkaConfig.SetKey("group.id", "om-sink")

	producerKafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	_ = producerKafkaConfig.SetKey("group.id", "om-sink-deadletter")

	sinkConfig := sink.SinkConfig{
		Context:             context.Background(),
		Logger:              logger,
		Storage:             storage,
		Deduplicator:        deduplicator,
		ConsumerKafkaConfig: consumerKafkaConfig,
		ProducerKafkaConfig: producerKafkaConfig,
		MinCommitCount:      config.Sink.MinCommitCount,
		MaxCommitWait:       config.Sink.MaxCommitWait,
		NamespaceRefetch:    config.Sink.NamespaceRefetch,
	}

	return sink.NewSink(&sinkConfig)
}
