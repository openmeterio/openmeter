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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
)

const (
	DefaultShutdownTimeout = 5 * time.Second
)

func NewTelemetryResource(metadata Metadata) *resource.Resource {
	extraResources, _ := resource.New(
		// TODO: use the globally available context here?
		context.Background(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName(metadata.ServiceName),
			semconv.ServiceVersion(metadata.Version),
			semconv.DeploymentEnvironment(metadata.Environment),
		),
	)

	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	return res
}

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

func NewMeter(meterProvider metric.MeterProvider, metadata Metadata) metric.Meter {
	return meterProvider.Meter(metadata.OpenTelemetryName)
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

func NewTracer(tracerProvider trace.TracerProvider, metadata Metadata) trace.Tracer {
	return tracerProvider.Tracer(metadata.OpenTelemetryName)
}

func NewDefaultTextMapPropagator() propagation.TextMapPropagator {
	return propagation.TraceContext{}
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

func NewTelemetryRouterHook(meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) func(chi.Router) {
	return func(r chi.Router) {
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
				otelhttp.WithMeterProvider(meterProvider),
				otelhttp.WithTracerProvider(tracerProvider),
			)
		})
	}
}
