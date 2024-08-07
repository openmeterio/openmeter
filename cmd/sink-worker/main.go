package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"

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
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	watermillkafka "github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var otelName string = "openmeter.io/sink-worker"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

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
		slog.Error("failed to read configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var conf config.Configuration
	err = v.Unmarshal(&conf)
	if err != nil {
		slog.Error("failed to unmarshal configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = conf.Validate()
	if err != nil {
		slog.Error("invalid configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	extraResources, _ := resource.New(
		ctx,
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("openmeter-sink-worker"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(conf.Environment),
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
	otelMeterProvider, err := conf.Telemetry.Metrics.NewMeterProvider(ctx, res)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry Metrics provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err = otelMeterProvider.Shutdown(ctx); err != nil {
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
		if err = otelTracerProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down tracer provider", slog.String("error", err.Error()))
		}
	}()

	otel.SetTracerProvider(otelTracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracer := otelTracerProvider.Tracer(otelName)

	// Configure health checker
	healthChecker := health.New(health.WithCheckListeners(gosundheit.NewLogger(logger.With(slog.String("component", "healthcheck")))))
	handler := healthhttp.HandleHealthJSON(healthChecker)
	telemetryRouter.Handle("/healthz", handler)

	// Kubernetes style health checks
	telemetryRouter.HandleFunc("/healthz/live", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	telemetryRouter.Handle("/healthz/ready", handler)

	logger.Info("starting OpenMeter sink worker", "config", map[string]string{
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	var group run.Group

	// initialize system event producer
	ingestEventFlushHandler, err := initIngestEventPublisher(ctx, logger, conf, metricMeter)
	if err != nil {
		logger.Error("failed to initialize event publisher", "error", err)
		os.Exit(1)
	}

	// Initialize meter repository
	meterRepository := meter.NewInMemoryRepository(slicesx.Map(conf.Meters, func(meter *models.Meter) models.Meter {
		return *meter
	}))

	// Initialize sink worker
	sink, err := initSink(conf, logger, metricMeter, tracer, meterRepository, ingestEventFlushHandler)
	if err != nil {
		logger.Error("failed to initialize sink worker", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = sink.Close()
	}()

	// Set up telemetry server
	server := &http.Server{
		Addr:    conf.Telemetry.Address,
		Handler: telemetryRouter,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	defer func() {
		_ = server.Close()
	}()

	// Add sink worker to run group
	group.Add(
		func() error { return sink.Run(ctx) },
		func(err error) { _ = sink.Close() },
	)

	// Add telemetry server to run group
	group.Add(
		func() error { return server.ListenAndServe() },
		func(err error) { _ = server.Shutdown(ctx) },
	)

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	// Run actors
	err = group.Run()

	if e := (run.SignalError{}); errors.As(err, &e) {
		logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func initIngestEventPublisher(ctx context.Context, logger *slog.Logger, conf config.Configuration, metricMeter metric.Meter) (flushhandler.FlushEventHandler, error) {
	if !conf.Events.Enabled {
		return nil, nil
	}

	eventDriver, err := watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:     otelName,
			Logger:       logger,
			DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
			MetricMeter:  metricMeter,
		},

		ProvisionTopics: []watermillkafka.AutoProvisionTopic{
			{
				Topic:         conf.Events.IngestEvents.Topic,
				NumPartitions: int32(conf.Events.IngestEvents.AutoProvision.Partitions),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	eventPublisher, err := eventbus.New(eventbus.Options{
		Publisher:              eventDriver,
		Config:                 conf.Events,
		Logger:                 logger,
		MarshalerTransformFunc: watermillkafka.AddPartitionKeyFromSubject,
	})
	if err != nil {
		return nil, err
	}

	flushHandlerMux := flushhandler.NewFlushEventHandlers()
	// We should only close the producer once the ingest events are fully processed
	flushHandlerMux.OnDrainComplete(func() {
		logger.Info("shutting down kafka producer")
		if err := eventDriver.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
		}
	})

	ingestNotificationHandler, err := ingestnotification.NewHandler(logger, metricMeter, eventPublisher, ingestnotification.HandlerConfig{
		MaxEventsInBatch: conf.Sink.IngestNotifications.MaxEventsInBatch,
	})
	if err != nil {
		return nil, err
	}

	flushHandlerMux.AddHandler(ingestNotificationHandler)
	return flushHandlerMux, nil
}

func initSink(config config.Configuration, logger *slog.Logger, metricMeter metric.Meter, tracer trace.Tracer, meterRepository meter.Repository, flushHandler flushhandler.FlushEventHandler) (*sink.Sink, error) {
	clickhouseClient, err := clickhouse.Open(config.Aggregation.ClickHouse.GetClientOptions())
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

	consumerKafkaConfig := config.Ingest.Kafka.CreateKafkaConfig()
	_ = consumerKafkaConfig.SetKey("group.id", config.Sink.GroupId)
	_ = consumerKafkaConfig.SetKey("session.timeout.ms", 6000)
	_ = consumerKafkaConfig.SetKey("enable.auto.commit", true)
	_ = consumerKafkaConfig.SetKey("enable.auto.offset.store", false)
	_ = consumerKafkaConfig.SetKey("go.application.rebalance.enable", true)
	// Used when offset retention resets the offset. In this case we want to consume from the latest offset as everything before should be already processed.
	_ = consumerKafkaConfig.SetKey("auto.offset.reset", "latest")
	// Guarantees an assignment that is maximally balanced while preserving as many existing partition assignments as possible.
	_ = consumerKafkaConfig.SetKey("partition.assignment.strategy", "cooperative-sticky")

	consumer, err := kafka.NewConsumer(&consumerKafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka consumer: %s", err)
	}

	// Enable Kafka client logging
	go pkgkafka.ConsumeLogChannel(consumer, logger.WithGroup("kafka").WithGroup("consumer"))

	sinkConfig := sink.SinkConfig{
		Logger:            logger,
		Tracer:            tracer,
		MetricMeter:       metricMeter,
		MeterRepository:   meterRepository,
		Storage:           storage,
		Deduplicator:      deduplicator,
		Consumer:          consumer,
		MinCommitCount:    config.Sink.MinCommitCount,
		MaxCommitWait:     config.Sink.MaxCommitWait,
		NamespaceRefetch:  config.Sink.NamespaceRefetch,
		FlushEventHandler: flushHandler,
	}

	return sink.NewSink(sinkConfig)
}
