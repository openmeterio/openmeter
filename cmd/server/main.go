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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerrepository "github.com/openmeterio/openmeter/openmeter/customer/repository"
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
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry/startup"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

const (
	defaultShutdownTimeout = 5 * time.Second
	otelName               = "openmeter.io/backend"
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
		errorsx.NewContextHandler(errorsx.NewAppHandler(errorsx.NewSlogHandler(logger))),
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

	entClient := app.EntClient

	if err := startup.DB(ctx, conf.Postgres, entClient); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	logger.Info("Postgres client initialized")

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

	if entClient != nil {
		var customerRepo customer.Repository
		customerRepo, err = customerrepository.New(customerrepository.Config{
			Client: entClient,
			Logger: logger.WithGroup("customer.postgres"),
		})
		if err != nil {
			logger.Error("failed to initialize customer repository", "error", err)
			os.Exit(1)
		}

		customerService, err = customer.NewService(customer.ServiceConfig{
			Repository: customerRepo,
		})
		if err != nil {
			logger.Error("failed to initialize customer service", "error", err)
			os.Exit(1)
		}
	}

	// Initialize Notification
	var notificationService notification.Service

	if conf.Notification.Enabled {
		if !conf.Entitlements.Enabled {
			logger.Error("failed to initialize notification service: entitlements must be enabled")
			os.Exit(1)
		}

		// CreatingPG client is done as part of entitlements initialization
		if entClient == nil {
			logger.Error("failed to initialize notification service: postgres client is not initialized")
			os.Exit(1)
		}

		var notificationRepo notification.Repository
		notificationRepo, err = notificationrepository.New(notificationrepository.Config{
			Client: entClient,
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

	s, err := server.NewServer(&server.Config{
		RouterConfig: router.Config{
			NamespaceManager:    app.NamespaceManager,
			StreamingConnector:  app.StreamingConnector,
			IngestHandler:       ingestHandler,
			Meters:              app.MeterRepository,
			PortalTokenStrategy: portalTokenStrategy,
			PortalCORSEnabled:   conf.Portal.CORS.Enabled,
			ErrorHandler:        errorsx.NewAppHandler(errorsx.NewSlogHandler(logger)),
			// deps
			Customer:                    customerService,
			DebugConnector:              debugConnector,
			FeatureConnector:            entitlementConnRegistry.Feature,
			EntitlementConnector:        entitlementConnRegistry.Entitlement,
			EntitlementBalanceConnector: entitlementConnRegistry.MeteredEntitlement,
			GrantConnector:              entitlementConnRegistry.Grant,
			GrantRepo:                   entitlementConnRegistry.GrantRepo,
			Notification:                notificationService,
			// modules
			EntitlementsEnabled: conf.Entitlements.Enabled,
			NotificationEnabled: conf.Notification.Enabled,
		},
		RouterHook: func(r chi.Router) {
			r.Use(func(h http.Handler) http.Handler {
				return otelhttp.NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						h.ServeHTTP(w, r)

						routePattern := chi.RouteContext(r.Context()).RoutePattern()

						span := trace.SpanFromContext(r.Context())
						span.SetName(routePattern)
						span.SetAttributes(semconv.HTTPTarget(r.URL.String()), semconv.HTTPRoute(routePattern))

						labeler, ok := otelhttp.LabelerFromContext(r.Context())
						if ok {
							labeler.Add(semconv.HTTPRoute(routePattern))
						}
					}),
					"",
					otelhttp.WithMeterProvider(app.MeterProvider),
					otelhttp.WithTracerProvider(app.TracerProvider),
				)
			})
		},
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
		err := app.StreamingConnector.CreateMeter(ctx, app.NamespaceManager.GetDefaultNamespace(), meter)
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
