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

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thmeitz/ksqldb-go/net"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/ingest/httpingest"
	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/internal/server"
	"github.com/openmeterio/openmeter/internal/server/router"
	"github.com/openmeterio/openmeter/internal/streaming/kafka_connector"
)

// TODO: inject logger in main
func init() {
	var logger *slog.Logger
	// TODO NO_COLOR
	if os.Getenv("LOG_FORMAT") == "json" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{
			Level: slog.LevelDebug,
		}))
	}
	slog.SetDefault(logger)
}

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

	logger.Info("starting OpenMeter server", "config", config)

	const topic = "om_events"

	// Initialize schema
	schemaRegistry, err := schemaregistry.NewClient(schemaregistry.NewConfig(config.Ingest.Kafka.SchemaRegistry))
	if err != nil {
		logger.Error("init schema registry client: %v", err)
		os.Exit(1)
	}

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.Ingest.Kafka.Broker,
	})
	if err != nil {
		logger.Error("init Kafka producer: %v", err)
		os.Exit(1)
	}

	defer producer.Flush(30 * 1000)
	defer producer.Close()

	slog.Debug("connected to Kafka")

	schema, err := kafkaingest.NewSchema(schemaRegistry, topic)
	if err != nil {
		logger.Error("init schema: %v", err)
		os.Exit(1)
	}

	collector := kafkaingest.Collector{
		Producer: producer,
		Topic:    topic,
		Schema:   schema,
	}

	// TODO: config file (https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md)
	connector, err := kafka_connector.NewKafkaConnector(&kafka_connector.KafkaConnectorConfig{
		Kafka: &kafka.ConfigMap{
			"bootstrap.servers": config.Ingest.Kafka.Broker,
		},
		KsqlDB: &net.Options{
			BaseUrl:   config.Processor.KSQLDB.URL,
			AllowHTTP: true,
		},
		SchemaRegistry: schemaregistry.NewConfig(config.Ingest.Kafka.SchemaRegistry),
		EventsTopic:    topic,
		Partitions:     config.Ingest.Kafka.Partitions,
	})
	if err != nil {
		slog.Error("failed to create streaming connector", "error", err)
		os.Exit(1)
	}
	defer connector.Close()

	slog.Info("kafka connector successfully initialized")

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			StreamingConnector: connector,
			IngestHandler: httpingest.Handler{
				Collector: collector,
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
		err := connector.Init(meter)
		if err != nil {
			slog.Warn("failed to initialize meter", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("meters successfully initialized", "count", len(config.Meters))

	var group run.Group

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

	group.Add(kafkaGroup(context.Background(), producer, logger))

	// Setup signal handler
	group.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		slog.Info("received signal; shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		slog.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

func kafkaGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger) (execute func() error, interrupt func(error)) {
	ctx, cancel := context.WithCancel(ctx)
	return func() error {
			for {
				select {
				case e := <-producer.Events():
					switch ev := e.(type) {
					case *kafka.Message:
						// The message delivery report, indicating success or
						// permanent failure after retries have been exhausted.
						// Application level retries won't help since the client
						// is already configured to do that.
						m := ev
						if m.TopicPartition.Error != nil {
							logger.Error("kafka delivery failed", "error", m.TopicPartition.Error)
						} else {
							logger.Debug("kafka message delivered", "topic", *m.TopicPartition.Topic, "partition", m.TopicPartition.Partition, "offset", m.TopicPartition.Offset)
						}
					case kafka.Error:
						// Generic client instance-level errors, such as
						// broker connection failures, authentication issues, etc.
						//
						// These errors should generally be considered informational
						// as the underlying client will automatically try to
						// recover from any errors encountered, the application
						// does not need to take action on them.
						logger.Error("kafka error", "error", ev)
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		func(error) {
			cancel()
		}
}
