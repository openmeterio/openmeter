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
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/debug"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/pkg/errorsx"
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

	var group run.Group

	// TODO: move kafkaingest.KafkaProducerGroup to pkg/kafka
	// TODO: move to .... somewhere else?
	group.Add(kafkaingest.KafkaProducerGroup(ctx, app.KafkaProducer, logger, app.KafkaMetrics))

	// Initialize Namespace
	err = initNamespace(app.NamespaceManager, logger)
	if err != nil {
		logger.Error("failed to initialize namespace", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP Ingest handler
	ingestService := ingest.Service{
		Collector: app.IngestCollector,
		Logger:    logger,
	}
	ingestHandler := ingestdriver.NewIngestEventsHandler(
		ingestService.IngestEvents,
		namespacedriver.StaticNamespaceDecoder(app.NamespaceManager.GetDefaultNamespace()),
		nil,
		errorsx.NewSlogHandler(logger),
	)

	// Initialize portal
	var portalTokenStrategy *authenticator.PortalTokenStrategy
	if conf.Portal.Enabled {
		portalTokenStrategy, err = authenticator.NewPortalTokenStrategy(conf.Portal.TokenSecret, conf.Portal.TokenExpiration)
		if err != nil {
			logger.Error("failed to initialize portal token strategy", "error", err)
			os.Exit(1)
		}
	}

	// Initialize debug connector
	debugConnector := debug.NewDebugConnector(app.StreamingConnector)

	// Migrate database
	if err := app.Migrate(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Provision sandbox app
	if conf.Apps.Enabled {
		err = app.AppSandboxProvisioner()
		if err != nil {
			logger.Error("failed to provision sandbox app", "error", err)
			os.Exit(1)
		}
	}

	if conf.Billing.Enabled {
		err = app.Billing.ProvisionDefaultBillingProfile(ctx, app.NamespaceManager.GetDefaultNamespace())
		if err != nil {
			logger.Error("failed to provision default billing profile", "error", err)
			os.Exit(1)
		}
	}

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			NamespaceManager:    app.NamespaceManager,
			StreamingConnector:  app.StreamingConnector,
			IngestHandler:       ingestHandler,
			Meters:              app.MeterRepository,
			PortalTokenStrategy: portalTokenStrategy,
			PortalCORSEnabled:   conf.Portal.CORS.Enabled,
			ErrorHandler:        errorsx.NewSlogHandler(logger),
			// deps
			App:                         app.App,
			AppStripe:                   app.AppStripe,
			Billing:                     app.Billing,
			Customer:                    app.Customer,
			DebugConnector:              debugConnector,
			EntitlementBalanceConnector: app.EntitlementRegistry.MeteredEntitlement,
			EntitlementConnector:        app.EntitlementRegistry.Entitlement,
			SubscriptionService:         app.Subscription.Service,
			SubscriptionWorkflowService: app.Subscription.WorkflowService,
			SubscriptionPlanAdapter:     app.SubscriptionPlanAdapter,
			SubscriptionChangeService:   app.SubscriptionChangeService,
			Logger:                      logger,
			FeatureConnector:            app.EntitlementRegistry.Feature,
			GrantConnector:              app.EntitlementRegistry.Grant,
			GrantRepo:                   app.EntitlementRegistry.GrantRepo,
			Notification:                app.Notification,
			Plan:                        app.Plan,
			// modules
			ProductCatalogEnabled: conf.ProductCatalog.Enabled,
			AppsEnabled:           conf.Apps.Enabled,
			BillingEnabled:        conf.Billing.Enabled,
		},
		RouterHook: app.RouterHook,
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

	for _, meter := range conf.Meters {
		err := app.StreamingConnector.CreateMeter(ctx, app.NamespaceManager.GetDefaultNamespace(), *meter)
		if err != nil {
			slog.Warn("failed to initialize meter", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("meters successfully created", "count", len(conf.Meters))

	// Set up telemetry server
	{
		server := app.TelemetryServer

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) },
		)
	}

	// Set up server
	{
		server := &http.Server{
			Addr:    conf.Address,
			Handler: s,
		}
		defer server.Close()

		group.Add(
			func() error { return server.ListenAndServe() },
			func(err error) { _ = server.Shutdown(ctx) }, // TODO: context deadline
		)
	}

	// Setup signal handler
	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	if e := (run.SignalError{}); errors.As(err, &e) {
		slog.Info("received signal; shutting down", slog.String("signal", e.Signal.String()))
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
