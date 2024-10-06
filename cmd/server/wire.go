//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/google/wire"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
)

type Application struct {
	Logger           *slog.Logger
	MeterProvider    *sdkmetric.MeterProvider
	Meter            metric.Meter
	TracerProvider   *sdktrace.TracerProvider
	HealthChecker    health.Health
	TelemetryHandler TelemetryHandler
}

func InitializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		set,
		ProvideApplication,
	)

	return Application{}, nil, nil
}

func ProvideApplication(
	logger *slog.Logger,
	meterProvider *sdkmetric.MeterProvider,
	meter metric.Meter,
	tracerProvider *sdktrace.TracerProvider,
	healthChecker health.Health,
	telemetryHandler TelemetryHandler,
) Application {
	return Application{
		Logger:           logger,
		MeterProvider:    meterProvider,
		Meter:            meter,
		TracerProvider:   tracerProvider,
		HealthChecker:    healthChecker,
		TelemetryHandler: telemetryHandler,
	}
}

var set = wire.NewSet(
	config.Set,

	ProvideLogger,
	ProvideOtelResource,
	ProvideOtelMeterProvider,
	ProvideOtelMeter,
	ProvideOtelTracerProvider,
	ProvideHealthChecker,
	ProvideTelemetryHandler,
)

func ProvideLogger(conf config.LogTelemetryConfiguration, res *resource.Resource) *slog.Logger {
	logger := slog.New(slogmulti.Pipe(
		otelslog.NewHandler,
		contextx.NewLogHandler,
		operation.NewLogHandler,
	).Handler(conf.NewHandler(os.Stdout)))

	logger = otelslog.WithResource(logger, res)

	return logger
}

func ProvideOtelResource(conf config.Configuration) *resource.Resource {
	extraResources, _ := resource.New(
		context.Background(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("openmeter"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(conf.Environment),
		),
	)

	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	return res
}

func ProvideOtelMeterProvider(ctx context.Context, conf config.MetricsTelemetryConfig, res *resource.Resource, logger *slog.Logger) (*sdkmetric.MeterProvider, func(), error) {
	meterProvider, err := conf.NewMeterProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry Metrics provider: %w", err)
	}

	return meterProvider, func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := meterProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down meter provider", slog.String("error", err.Error()))
		}
	}, nil
}

func ProvideOtelMeter(meterProvider *sdkmetric.MeterProvider) metric.Meter {
	return meterProvider.Meter(otelName)
}

func ProvideOtelTracerProvider(ctx context.Context, conf config.TraceTelemetryConfig, res *resource.Resource, logger *slog.Logger) (*sdktrace.TracerProvider, func(), error) {
	tracerProvider, err := conf.NewTracerProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry Trace provider: %w", err)
	}

	return tracerProvider, func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down tracer provider", slog.String("error", err.Error()))
		}
	}, nil
}

func ProvideHealthChecker(logger *slog.Logger) health.Health {
	return health.New(health.WithCheckListeners(gosundheit.NewLogger(logger.With(slog.String("component", "healthcheck")))))
}

type TelemetryHandler http.Handler

func ProvideTelemetryHandler(metricsConf config.MetricsTelemetryConfig, healthChecker health.Health) TelemetryHandler {
	router := chi.NewRouter()
	router.Mount("/debug", middleware.Profiler())

	if metricsConf.Exporters.Prometheus.Enabled {
		router.Handle("/metrics", promhttp.Handler())
	}

	handler := healthhttp.HandleHealthJSON(healthChecker)
	router.Handle("/healthz", handler)

	// Kubernetes style health checks
	router.HandleFunc("/healthz/live", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	router.Handle("/healthz/ready", handler)

	return router
}
