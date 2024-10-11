package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/ClickHouse/clickhouse-go/v2"
	confluentkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/oklog/run"
	"github.com/sagikazarmark/slog-shim"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

var otelName string = "openmeter.io/sink-worker"

func main() {
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

	logger := initializeLogger(conf)

	app, cleanup, err := initializeApplication(ctx, conf, logger)
	if err != nil {
		logger.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	app.SetGlobals()

	tracer := app.TracerProvider.Tracer(otelName)

	logger.Info("starting OpenMeter sink worker", "config", map[string]string{
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	var group run.Group

	// initialize system event producer
	ingestEventFlushHandler, err := initIngestEventPublisher(ctx, logger, conf, app.Meter, app.TopicProvisioner)
	if err != nil {
		logger.Error("failed to initialize event publisher", "error", err)
		os.Exit(1)
	}

	// Initialize sink worker
	sink, err := initSink(conf, logger, app.Meter, tracer, app.MeterRepository, ingestEventFlushHandler)
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
	err = group.Run()

	if e := (run.SignalError{}); errors.As(err, &e) {
		logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func initIngestEventPublisher(ctx context.Context, logger *slog.Logger, conf config.Configuration, metricMeter metric.Meter, topicProvisioner pkgkafka.TopicProvisioner) (flushhandler.FlushEventHandler, error) {
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
		ProvisionTopics: []pkgkafka.TopicConfig{
			{
				Name:       conf.Events.IngestEvents.Topic,
				Partitions: conf.Events.IngestEvents.AutoProvision.Partitions,
			},
		},
		TopicProvisioner: topicProvisioner,
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

	// Initialize Kafka consumer

	consumerConfig := config.Sink.Kafka.AsConsumerConfig()

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

	topicResolver, err := topicresolver.NewNamespacedTopicResolver(config.Ingest.Kafka.EventsTopicTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic name resolver: %w", err)
	}

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
		TopicResolver:     topicResolver,
	}

	return sink.NewSink(sinkConfig)
}
