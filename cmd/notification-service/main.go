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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry/startup"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

const (
	otelName = "openmeter.io/notification-service"
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

	// TODO: move to global initializer
	slog.SetDefault(logger)

	// TODO: move to global initializer
	otel.SetMeterProvider(app.MeterProvider)
	otel.SetTracerProvider(app.TracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Validate service prerequisites

	if !conf.Events.Enabled {
		logger.Error("events are disabled, exiting")
		os.Exit(1)
	}

	var group run.Group

	entClient := app.EntClient

	// Run database schema creation
	if err := startup.DB(ctx, conf.Postgres, entClient); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	logger.Info("Postgres client initialized")

	// Create  subscriber
	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            wmBrokerConfiguration(conf, logger, app.Meter),
		ConsumerGroupName: conf.Notification.Consumer.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create publisher
	eventPublisherDriver, err := initEventPublisherDriver(ctx, logger, conf, app.Meter)
	if err != nil {
		logger.Error("failed to initialize event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		// We are using sync producer, so it is fine to close this as a last step
		if err := eventPublisherDriver.Close(); err != nil {
			logger.Error("failed to close kafka producer", slog.String("error", err.Error()))
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
	entitlementConnRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     entClient,
		StreamingConnector: app.StreamingConnector,
		MeterRepository:    app.MeterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})

	// Dependencies: notification
	notificationRepo, err := notificationrepository.New(notificationrepository.Config{
		Client: entClient,
		Logger: logger.WithGroup("notification.postgres"),
	})
	if err != nil {
		logger.Error("failed to initialize notification repository", "error", err)
		os.Exit(1)
	}

	notificationWebhook, err := notificationwebhook.New(notificationwebhook.Config{
		SvixConfig:              conf.Svix,
		RegistrationTimeout:     conf.Notification.Webhook.EventTypeRegistrationTimeout,
		SkipRegistrationOnError: conf.Notification.Webhook.SkipEventTypeRegistrationOnError,
		Logger:                  logger.WithGroup("notification.webhook"),
	})
	if err != nil {
		logger.Error("failed to initialize notification repository", "error", err)
		os.Exit(1)
	}

	notificationService, err := notificationservice.New(notificationservice.Config{
		Repository:       notificationRepo,
		Webhook:          notificationWebhook,
		FeatureConnector: entitlementConnRegistry.Feature,
		Logger:           logger.With(slog.String("subsystem", "notification")),
	})
	if err != nil {
		logger.Error("failed to initialize notification service", "error", err)
		os.Exit(1)
	}

	// Initialize consumer
	consumerOptions := consumer.Options{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   eventPublisherDriver,
			Logger:      logger,
			MetricMeter: app.Meter,

			Config: conf.Notification.Consumer,
		},
		Marshaler: eventPublisher.Marshaler(),

		Notification: notificationService,

		Logger: logger,
	}

	notifictionConsumer, err := consumer.New(consumerOptions)
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

	// Notification service consumer
	group.Add(
		func() error { return notifictionConsumer.Run(ctx) },
		func(err error) { _ = notifictionConsumer.Close() },
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

func wmBrokerConfiguration(conf config.Configuration, logger *slog.Logger, metricMeter metric.Meter) kafka.BrokerOptions {
	return kafka.BrokerOptions{
		KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
		ClientID:     otelName,
		Logger:       logger,
		MetricMeter:  metricMeter,
		DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
	}
}

func initEventPublisherDriver(ctx context.Context, logger *slog.Logger, conf config.Configuration, metricMeter metric.Meter) (message.Publisher, error) {
	provisionTopics := []watermillkafka.AutoProvisionTopic{}
	if conf.Notification.Consumer.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, watermillkafka.AutoProvisionTopic{
			Topic:         conf.Notification.Consumer.DLQ.Topic,
			NumPartitions: int32(conf.Notification.Consumer.DLQ.AutoProvision.Partitions),
			Retention:     conf.BalanceWorker.DLQ.AutoProvision.Retention,
		})
	}

	return watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker:          wmBrokerConfiguration(conf, logger, metricMeter),
		ProvisionTopics: provisionTopics,
	})
}
