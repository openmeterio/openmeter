package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	health "github.com/AppsFlyer/go-sundheit"
	healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-slog/otelslog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogmulti "github.com/samber/slog-multi"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
)

const (
	DefaultShutdownTimeout = 5 * time.Second
)

func NewLogger(conf config.LogTelemetryConfiguration, res *resource.Resource) *slog.Logger {
	logger := slog.New(slogmulti.Pipe(
		otelslog.NewHandler,
		contextx.NewLogHandler,
		operation.NewLogHandler,
	).Handler(conf.NewHandler(os.Stdout)))

	logger = otelslog.WithResource(logger, res)

	return logger
}

func NewMeterProvider(ctx context.Context, conf config.MetricsTelemetryConfig, res *resource.Resource, logger *slog.Logger) (*sdkmetric.MeterProvider, func(), error) {
	meterProvider, err := conf.NewMeterProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry Metrics provider: %w", err)
	}

	return meterProvider, func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
		defer cancel()

		if err := meterProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down meter provider", slog.String("error", err.Error()))
		}
	}, nil
}

func NewTracerProvider(ctx context.Context, conf config.TraceTelemetryConfig, res *resource.Resource, logger *slog.Logger) (*sdktrace.TracerProvider, func(), error) {
	tracerProvider, err := conf.NewTracerProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry Trace provider: %w", err)
	}

	return tracerProvider, func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Error("shutting down tracer provider", slog.String("error", err.Error()))
		}
	}, nil
}

func NewHealthChecker(logger *slog.Logger) health.Health {
	return health.New(health.WithCheckListeners(gosundheit.NewLogger(logger.With(slog.String("component", "healthcheck")))))
}

type TelemetryHandler http.Handler

func NewTelemetryHandler(metricsConf config.MetricsTelemetryConfig, healthChecker health.Health) TelemetryHandler {
	router := chi.NewRouter()
	router.Mount("/debug", middleware.Profiler())

	if metricsConf.Exporters.Prometheus.Enabled {
		router.Handle("/metrics", promhttp.Handler())
	}

	// Health
	{
		handler := healthhttp.HandleHealthJSON(healthChecker)
		router.Handle("/healthz", handler)

		// Kubernetes style health checks
		router.HandleFunc("/healthz/live", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		router.Handle("/healthz/ready", handler)
	}

	return router
}

type TelemetryServer = *http.Server

func NewTelemetryServer(conf config.TelemetryConfig, handler TelemetryHandler) (TelemetryServer, func()) {
	server := &http.Server{
		Addr:    conf.Address,
		Handler: handler,
	}

	return server, func() { server.Close() }
}
