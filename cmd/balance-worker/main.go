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
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
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

	if err := app.Migrate(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	wmSubscriber, err := watermillkafka.NewSubscriber(watermillkafka.SubscriberOptions{
		Broker:            app.BrokerOptions,
		ConsumerGroupName: conf.BalanceWorker.ConsumerGroupName,
	})
	if err != nil {
		logger.Error("failed to initialize Kafka subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Dependencies: entitlement
	entitlementConnectors := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     app.EntClient,
		StreamingConnector: app.StreamingConnector,
		MeterRepository:    app.MeterRepository,
		Logger:             logger,
		Publisher:          app.EventPublisher,
	})

	// Initialize worker
	workerOptions := balanceworker.WorkerOptions{
		SystemEventsTopic: conf.Events.SystemEvents.Topic,
		IngestEventsTopic: conf.Events.IngestEvents.Topic,

		Router: router.Options{
			Subscriber:  wmSubscriber,
			Publisher:   app.MessagePublisher,
			Logger:      logger,
			MetricMeter: app.Meter,

			Config: conf.BalanceWorker.ConsumerConfiguration,
		},

		EventBus: app.EventPublisher,

		Entitlement: entitlementConnectors,
		Repo:        entitlementpgadapter.NewPostgresEntitlementRepo(app.EntClient),

		Logger: logger,
	}

	worker, err := balanceworker.New(workerOptions)
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
