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
	apppkg "github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/debug"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationrepository "github.com/openmeterio/openmeter/openmeter/notification/repository"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	plan "github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
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

		cleanup()

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

	debugConnector := debug.NewDebugConnector(app.StreamingConnector)
	entitlementConnRegistry := &registry.Entitlement{}

	if err := app.Migrate(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	if conf.Entitlements.Enabled {
		entitlementConnRegistry = registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
			DatabaseClient:     app.EntClient,
			StreamingConnector: app.StreamingConnector,
			MeterRepository:    app.MeterRepository,
			Logger:             logger,
			Publisher:          app.EventPublisher,
		})
	}

	// Initialize Customer
	var customerService customer.CustomerService

	if app.EntClient != nil {
		var customerAdapter customer.Adapter
		customerAdapter, err = customeradapter.New(customeradapter.Config{
			Client: app.EntClient,
			Logger: logger.WithGroup("customer.postgres"),
		})
		if err != nil {
			logger.Error("failed to initialize customer repository", "error", err)
			os.Exit(1)
		}

		customerService, err = customerservice.New(customerservice.Config{
			Adapter: customerAdapter,
		})
		if err != nil {
			logger.Error("failed to initialize customer service", "error", err)
			os.Exit(1)
		}
	}

	// Initialize Secret
	secretService, err := secretservice.New(secretservice.Config{
		Adapter: secretadapter.New(),
	})
	if err != nil {
		logger.Error("failed to initialize secret service", "error", err)
		os.Exit(1)
	}

	// Initialize App
	var appService apppkg.Service

	if conf.Apps.Enabled {
		var appAdapter apppkg.Adapter
		appAdapter, err = appadapter.New(appadapter.Config{
			Client:  app.EntClient,
			BaseURL: conf.StripeApp.IncomingWebhook.BaseURL,
		})
		if err != nil {
			logger.Error("failed to initialize app repository", "error", err)
			os.Exit(1)
		}

		appService, err = appservice.New(appservice.Config{
			Adapter: appAdapter,
		})
		if err != nil {
			logger.Error("failed to initialize app service", "error", err)
			os.Exit(1)
		}
	}

	// Initialize AppStripe
	var appStripeService appstripe.Service

	if conf.Apps.Enabled {
		var appStripeAdapter appstripe.Adapter
		appStripeAdapter, err = appstripeadapter.New(appstripeadapter.Config{
			Client:          app.EntClient,
			AppService:      appService,
			CustomerService: customerService,
			SecretService:   secretService,
		})
		if err != nil {
			logger.Error("failed to initialize app stripe repository", "error", err)
			os.Exit(1)
		}

		appStripeService, err = appstripeservice.New(appstripeservice.Config{
			Adapter: appStripeAdapter,
		})
		if err != nil {
			logger.Error("failed to initialize app stripe service", "error", err)
			os.Exit(1)
		}
	}

	// Initialize AppSandbox
	if conf.Apps.Enabled {
		_, err = appsandbox.NewFactory(appsandbox.Config{
			AppService: appService,
		})
		if err != nil {
			logger.Error("failed to initialize app sandbox factory", "error", err)
			os.Exit(1)
		}

		app, err := appsandbox.AutoProvision(ctx, appsandbox.AutoProvisionInput{
			Namespace:  app.NamespaceManager.GetDefaultNamespace(),
			AppService: appService,
		})
		if err != nil {
			logger.Error("failed to auto-provision sandbox app", "error", err)
			os.Exit(1)
		}

		logger.Info("sandbox app auto-provisioned", "app_id", app.GetID().ID)
	}

	// Initialize Notification
	var notificationService notification.Service

	if conf.Notification.Enabled {
		if !conf.Entitlements.Enabled {
			logger.Error("failed to initialize notification service: entitlements must be enabled")
			os.Exit(1)
		}

		// CreatingPG client is done as part of entitlements initialization
		if app.EntClient == nil {
			logger.Error("failed to initialize notification service: postgres client is not initialized")
			os.Exit(1)
		}

		var notificationRepo notification.Repository
		notificationRepo, err = notificationrepository.New(notificationrepository.Config{
			Client: app.EntClient,
			Logger: logger.WithGroup("notification.postgres"),
		})
		if err != nil {
			logger.Error("failed to initialize notification repository", "error", err)
			os.Exit(1)
		}

		var notificationWebhook notificationwebhook.Handler
		notificationWebhook, err = notificationwebhook.New(notificationwebhook.Config{
			SvixConfig:              conf.Svix,
			RegistrationTimeout:     conf.Notification.Webhook.EventTypeRegistrationTimeout,
			SkipRegistrationOnError: conf.Notification.Webhook.SkipEventTypeRegistrationOnError,
			Logger:                  logger.WithGroup("notification.webhook"),
		})
		if err != nil {
			logger.Error("failed to initialize notification webhook handler", "error", err)
			os.Exit(1)
		}

		notificationService, err = notificationservice.New(notificationservice.Config{
			Repository:       notificationRepo,
			Webhook:          notificationWebhook,
			FeatureConnector: entitlementConnRegistry.Feature,
			Logger:           logger.With(slog.String("subsystem", "notification")),
		})
		if err != nil {
			logger.Error("failed to initialize notification service", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err = notificationService.Close(); err != nil {
				logger.Error("failed to close notification service", "error", err)
			}
		}()
	}

	// Initialize plans
	var planService plan.Service
	if conf.ProductCatalog.Enabled {
		adapter, err := planadapter.New(planadapter.Config{
			Client: app.EntClient,
			Logger: logger.With("subsystem", "productcatalog.plan"),
		})
		if err != nil {
			logger.Error("failed to initialize plan adapter", "error", err)
			os.Exit(1)
		}

		planService, err = planservice.New(planservice.Config{
			Feature: entitlementConnRegistry.Feature,
			Adapter: adapter,
			Logger:  logger.With("subsystem", "productcatalog.plan"),
		})
		if err != nil {
			logger.Error("failed to initialize plan service", "error", err)
			os.Exit(1)
		}
	}

	// Initialize billing
	var billingService billing.Service
	if conf.Billing.Enabled {
		adapter, err := billingadapter.New(billingadapter.Config{
			Client: app.EntClient,
			Logger: logger.With("subsystem", "billing.adapter"),
		})
		if err != nil {
			logger.Error("failed to initialize billing adapter", "error", err)
			os.Exit(1)
		}

		billingService, err = billingservice.New(billingservice.Config{
			Adapter:            adapter,
			CustomerService:    customerService,
			AppService:         appService,
			Logger:             logger.With("subsystem", "billing.service"),
			FeatureService:     entitlementConnRegistry.Feature,
			MeterRepo:          app.MeterRepository,
			StreamingConnector: app.StreamingConnector,
		})
		if err != nil {
			logger.Error("failed to initialize billing service", "error", err)
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
			App:                         appService,
			AppStripe:                   appStripeService,
			Billing:                     billingService,
			Customer:                    customerService,
			DebugConnector:              debugConnector,
			EntitlementBalanceConnector: entitlementConnRegistry.MeteredEntitlement,
			EntitlementConnector:        entitlementConnRegistry.Entitlement,
			FeatureConnector:            entitlementConnRegistry.Feature,
			GrantConnector:              entitlementConnRegistry.Grant,
			GrantRepo:                   entitlementConnRegistry.GrantRepo,
			Notification:                notificationService,
			Plan:                        planService,
			// modules
			EntitlementsEnabled: conf.Entitlements.Enabled,
			NotificationEnabled: conf.Notification.Enabled,
			AppsEnabled:         conf.Apps.Enabled,
			BillingEnabled:      conf.Billing.Enabled,
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
