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
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
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

	// Add sink worker to run group
	group.Add(
		func() error { return app.Sink.Run(ctx) },
		func(err error) { _ = app.Sink.Close() },
	)

	// Set up telemetry server
	group.Add(
		func() error { return app.TelemetryServer.ListenAndServe() },
		func(err error) { _ = app.TelemetryServer.Shutdown(ctx) },
	)

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	// Run actors
	err = group.Run(run.WithReverseShutdownOrder())
	if err != nil {
		// Ignore HTTP server shutdown error
		if errors.Is(err, http.ErrServerClosed) {
			return
		}

		if e, ok := lo.ErrorsAs[*run.SignalError](err); ok {
			logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))

			return
		}

		logger.Error("application stopped due to error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
