package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry/startup"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

const (
	otelName = "openmeter.io/balance-worker"
)

func main() {
	v, flags := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook())), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)
	ctx := context.Background()

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
		panic(err)
	}

	var conf config.Configuration
	err = v.Unmarshal(&conf)
	if err != nil {
		panic(err)
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

	// Validate service prerequisites

	if !conf.Events.Enabled {
		logger.Error("events are disabled, exiting")
		os.Exit(1)
	}

	var group run.Group

	entClient := app.EntClient

	if err := startup.DB(ctx, conf.Postgres, entClient); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	logger.Info("Postgres client initialized")

	// Create  subscriber
	wmBrokerConfig := wmBrokerConfiguration(conf, logger, app.Meter)

	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            wmBrokerConfig,
		ConsumerGroupName: conf.BalanceWorker.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create publisher
	eventPublisherDriver, err := initEventPublisherDriver(ctx, wmBrokerConfig, conf, app.TopicProvisioner)
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
	entitlementConnectors := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     entClient,
		StreamingConnector: app.StreamingConnector,
		MeterRepository:    app.MeterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})

	// Initialize worker
	workerOptions := balanceworker.WorkerOptions{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		IngestEventsTopic: conf.Events.IngestEvents.Topic,

		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   eventPublisherDriver,
			Logger:      logger,
			MetricMeter: app.Meter,

			Config: conf.BalanceWorker.ConsumerConfiguration,
		},

		EventBus: eventPublisher,

		Entitlement: entitlementConnectors,
		Repo:        entitlementpgadapter.NewPostgresEntitlementRepo(entClient),

		Logger: logger,
	}

	worker, err := balanceworker.New(workerOptions)
	if err != nil {
		logger.Error("failed to initialize worker", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Run worker components

	// Set up telemetry server
	{
		server := app.TelemetryServer

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) },
		)
	}

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

func initEventPublisherDriver(ctx context.Context, broker watermillkafka.BrokerOptions, conf config.Configuration, topicProvisioner pkgkafka.TopicProvisioner) (message.Publisher, error) {
	var provisionTopics []pkgkafka.TopicConfig
	if conf.BalanceWorker.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.BalanceWorker.DLQ.Topic,
			Partitions:    conf.BalanceWorker.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.BalanceWorker.DLQ.AutoProvision.Retention),
		})
	}

	return watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker:           broker,
		ProvisionTopics:  provisionTopics,
		TopicProvisioner: topicProvisioner,
	})
}
