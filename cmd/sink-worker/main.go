package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"syscall"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/openmeterio/openmeter/pkg/log"
)

func main() {
	defer log.PanicLogger(log.WithExit)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	config.SetViperDefaults(v, flags)

	flags.String("config", "", "Configuration file")
	flags.Bool("version", false, "Show version information")
	flags.Bool("validate", false, "Validate configuration and exit")

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
		println("configuration error:")
		println(err.Error())
		os.Exit(1)
	}

	if v, _ := flags.GetBool("validate"); v {
		os.Exit(0)
	}

	app, cleanup, err := initializeApplication(ctx, conf)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)

		// Call cleanup function is may not set yet
		if cleanup != nil {
			cleanup()
		}

		os.Exit(1)
	}
	defer cleanup()

	app.SetGlobals()

	logger := app.Logger

	logger.Info("starting OpenMeter sink worker", "config", map[string]string{
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	var group run.Group

	// Initialize sink worker
	sink, err := initSink(
		conf,
		logger,
		app.Meter,
		app.Streaming,
		app.Tracer,
		app.TopicResolver,
		app.FlushHandler,
		app.MeterService,
	)
	if err != nil {
		logger.Error("failed to initialize sink worker", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = sink.Close()
	}()

	// Add sink worker to run group
	group.Add(
		func() error { return sink.Run(ctx) },
		func(err error) { _ = sink.Close() },
	)

	// Set up telemetry server
	{
		server := app.TelemetryServer

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) },
		)
	}

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	// Run actors
	err = group.Run(run.WithReverseShutdownOrder())

	if e := &(run.SignalError{}); errors.As(err, &e) {
		logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func initSink(
	conf config.Configuration,
	logger *slog.Logger,
	metricMeter metric.Meter,
	streaming streaming.Connector,
	tracer trace.Tracer,
	topicResolver *topicresolver.NamespacedTopicResolver,
	flushHandler flushhandler.FlushEventHandler,
	meterService meter.Service,
) (*sink.Sink, error) {
	var err error

	// Temporary: copy over sink storage settings
	// TODO: remove after config migration is over
	if conf.Sink.Storage.AsyncInsert {
		conf.Aggregation.AsyncInsert = conf.Sink.Storage.AsyncInsert
	}
	if conf.Sink.Storage.AsyncInsertWait {
		conf.Aggregation.AsyncInsertWait = conf.Sink.Storage.AsyncInsertWait
	}
	if conf.Sink.Storage.QuerySettings != nil {
		conf.Aggregation.InsertQuerySettings = conf.Sink.Storage.QuerySettings
	}

	// Initialize deduplicator if enabled
	var deduplicator dedupe.Deduplicator
	if conf.Sink.Dedupe.Enabled {
		deduplicator, err = conf.Sink.Dedupe.NewDeduplicator()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize deduplicator: %w", err)
		}
	}

	// Initialize storage
	storage, err := sink.NewClickhouseStorage(sink.ClickHouseStorageConfig{
		Streaming: streaming,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize Kafka consumer

	consumerConfig := conf.Sink.Kafka.AsConsumerConfig()

	// Override following Kafka consumer configuration parameters with hardcoded values as the Sink implementation relies on
	// these to be set to a specific value.
	consumerConfig.EnableAutoCommit = true
	consumerConfig.EnableAutoOffsetStore = false
	// Used when offset retention resets the offset. In this case we want to consume from the latest offset
	// as everything before should be already processed.
	consumerConfig.AutoOffsetReset = "latest"

	if err = consumerConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Kafka consumer configuration: %w", err)
	}

	logger.WithGroup("kafka").
		Debug("initializing Kafka consumer with group configuration",
			"group.id", consumerConfig.ConsumerGroupID,
			"group.instance.id", consumerConfig.ConsumerGroupInstanceID,
			"client.id", consumerConfig.ClientID,
			"session.timeout", consumerConfig.SessionTimeout.Duration().String(),
		)

	consumerConfigMap, err := consumerConfig.AsConfigMap()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Kafka configuration map: %w", err)
	}

	consumer, err := confluentkafka.NewConsumer(&consumerConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kafka consumer: %s", err)
	}

	// Enable Kafka client logging
	go pkgkafka.ConsumeLogChannel(consumer, logger.WithGroup("kafka").WithGroup("consumer"))

	sinkConfig := sink.SinkConfig{
		Logger:                  logger,
		Tracer:                  tracer,
		MetricMeter:             metricMeter,
		Storage:                 storage,
		Deduplicator:            deduplicator,
		Consumer:                consumer,
		MinCommitCount:          conf.Sink.MinCommitCount,
		MaxCommitWait:           conf.Sink.MaxCommitWait,
		MaxPollTimeout:          conf.Sink.MaxPollTimeout,
		FlushSuccessTimeout:     conf.Sink.FlushSuccessTimeout,
		DrainTimeout:            conf.Sink.DrainTimeout,
		NamespaceRefetch:        conf.Sink.NamespaceRefetch,
		FlushEventHandler:       flushHandler,
		TopicResolver:           topicResolver,
		NamespaceRefetchTimeout: conf.Sink.NamespaceRefetchTimeout,
		NamespaceTopicRegexp:    conf.Sink.NamespaceTopicRegexp,
		MeterRefetchInterval:    conf.Sink.MeterRefetchInterval,
		MeterService:            meterService,
	}

	return sink.NewSink(sinkConfig)
}
