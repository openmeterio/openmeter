package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"syscall"

	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/log"
)

func main() {
	defer log.PanicLogger(log.WithExit)

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

	// Initialize application
	app, cleanup, err := initializeApplication(ctx, conf)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)

		cleanup()

		os.Exit(1)
	}
	defer cleanup()

	app.SetGlobals()

	logger := app.Logger

	// Validate service prerequisites

	if err := app.Migrate(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Create  subscriber
	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            app.BrokerOptions,
		ConsumerGroupName: conf.Notification.Consumer.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize consumer
	consumerOptions := consumer.Options{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   app.MessagePublisher,
			Logger:      logger,
			MetricMeter: app.Meter,

			Config: conf.Notification.Consumer,
		},
		Marshaler: app.EventPublisher.Marshaler(),

		Notification: app.Notification,

		Logger: logger,
	}

	notifictionConsumer, err := consumer.New(consumerOptions)
	if err != nil {
		logger.Error("failed to initialize worker", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var group run.Group

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
	err = group.Run(run.WithReverseShutdownOrder())
	if e := &(run.SignalError{}); errors.As(err, &e) {
		logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}
