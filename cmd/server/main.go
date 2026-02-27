package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"syscall"

	"github.com/oklog/run"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/debug"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/pkg/errorsx"
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

	logger.Info("starting OpenMeter server", "config", map[string]string{
		"address":             conf.Address,
		"telemetry.address":   conf.Telemetry.Address,
		"ingest.kafka.broker": conf.Ingest.Kafka.Broker,
	})

	// Initialize Namespace
	err = initNamespace(app.NamespaceManager, logger)
	if err != nil {
		logger.Error("failed to initialize namespace", "error", err)
		os.Exit(1)
	}

	// Register Kafka Ingest Namespace Handler
	err = app.NamespaceManager.RegisterHandler(app.KafkaIngestNamespaceHandler)
	if err != nil {
		logger.Error("failed to register kafka ingest namespace handler", "error", err)
		os.Exit(1)
	}

	// Initialize debug connector
	debugConnector := debug.NewDebugConnector(app.StreamingConnector)

	// Migrate database
	if err := app.Migrate(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Provision sandbox app
	err = app.AppRegistry.SandboxProvisioner(ctx, app.NamespaceManager.GetDefaultNamespace())
	if err != nil {
		logger.Error("failed to provision sandbox app", "error", err)
		os.Exit(1)
	}

	err = app.Billing.ProvisionDefaultBillingProfile(ctx, app.NamespaceManager.GetDefaultNamespace())
	if err != nil {
		logger.Error("failed to provision default billing profile", "error", err)
		os.Exit(1)
	}

	// Create meters from config
	err = app.MeterConfigInitializer(ctx)
	if err != nil {
		logger.Error("failed to initialize meters from config", "error", err)
		os.Exit(1)
	}

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			Addon:                       app.Addon,
			App:                         app.AppRegistry.Service,
			AppStripe:                   app.AppRegistry.Stripe,
			AppCustomInvoicing:          app.AppRegistry.CustomInvoicing,
			Billing:                     app.Billing,
			BillingFeatureSwitches:      conf.Billing.FeatureSwitches,
			Customer:                    app.Customer,
			DebugConnector:              debugConnector,
			ErrorHandler:                errorsx.NewSlogHandler(logger),
			EntitlementBalanceConnector: app.EntitlementRegistry.MeteredEntitlement,
			EntitlementConnector:        app.EntitlementRegistry.Entitlement,
			FeatureConnector:            app.FeatureConnector,
			GrantConnector:              app.EntitlementRegistry.Grant,
			GrantRepo:                   app.EntitlementRegistry.GrantRepo,
			IngestService:               app.IngestService,
			LLMCostService:              app.LLMCostService,
			Logger:                      logger,
			MeterManageService:          app.MeterManageService,
			MeterEventService:           app.MeterEventService,
			NamespaceManager:            app.NamespaceManager,
			Notification:                app.Notification,
			Plan:                        app.Plan,
			PlanAddon:                   app.PlanAddon,
			PlanSubscriptionService:     app.Subscription.PlanSubscriptionService,
			Portal:                      app.Portal,
			PortalCORSEnabled:           conf.Portal.CORS.Enabled,
			ProgressManager:             app.ProgressManager,
			SubscriptionService:         app.Subscription.Service,
			SubscriptionWorkflowService: app.Subscription.WorkflowService,
			SubscriptionAddonService:    app.Subscription.SubscriptionAddonService,
			SubjectService:              app.SubjectService,
			StreamingConnector:          app.StreamingConnector,
		},
		RouterHooks:         lo.FromPtr(app.RouterHooks),
		PostAuthMiddlewares: app.PostAuthMiddlewares,
	})
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	s.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version": version,
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
		})
	})

	var group run.Group

	// Set up telemetry server
	{
		// Close telemetry server on exit. It drops all connections regardless of their states.
		defer func() {
			if err = app.TelemetryServer.Close(); err != nil {
				logger.Warn("failed to close telemetry server", "error", err)
			}
		}()

		telemetryServerRun := func() error {
			logger.Info("starting telemetry server", slog.String("address", conf.Telemetry.Address))

			return app.TelemetryServer.ListenAndServe()
		}

		telemetryServerShutdown := func(err error) {
			logger.Debug("shutting down telemetry server gracefully...", "error", err)

			if err = app.TelemetryServer.Shutdown(ctx); err != nil {
				logger.Warn("failed to shutdown telemetry server", "error", err)
			}
		}

		group.Add(telemetryServerRun, telemetryServerShutdown)
	}

	// Set up kafka ingest
	group.Add(kafkaingest.KafkaProducerGroup(ctx, app.KafkaProducer, logger, app.KafkaMetrics))

	// Set up server
	{
		apiServer := &http.Server{
			Addr:    conf.Address,
			Handler: s,
		}

		// Close API server on exit. It drops all connections regardless of their states.
		defer func() {
			if err = apiServer.Close(); err != nil {
				logger.Warn("failed to close API server", "error", err)
			}
		}()

		apiServerRun := func() error {
			logger.Info("starting API server", slog.String("address", conf.Address))

			return apiServer.ListenAndServe()
		}

		apiServerShutdown := func(err error) {
			logger.Debug("shutting down API server gracefully...", "error", err)

			if err = apiServer.Shutdown(ctx); err != nil {
				logger.Warn("failed to shutdown API server", "error", err)
			}
		}

		group.Add(apiServerRun, apiServerShutdown)
	}

	// Add service termination checker
	{
		terminationCheckerRun, terminationCheckerShutdown, err := common.NewTerminationCheckerActor(app.TerminationChecker, app.Logger)
		if err != nil {
			logger.Error("failed to initialize termination checker actor", "error", err)
		}

		group.Add(terminationCheckerRun, terminationCheckerShutdown)
	}

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	err = group.Run(run.WithReverseShutdownOrder())
	if e := &(run.SignalError{}); errors.As(err, &e) {
		logger.Info("received signal: shutting down", slog.String("signal", e.Signal.String()))
	} else if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("application stopped due to error", slog.String("error", err.Error()))
	}
}

func initNamespace(manager *namespace.Manager, logger *slog.Logger) error {
	logger.Debug("create default namespace")

	err := manager.CreateDefaultNamespace(context.Background())
	if err != nil {
		return fmt.Errorf("create default namespace: %v", err)
	}

	logger.Info("default namespace created")

	return nil
}
