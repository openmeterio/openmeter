package common

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
	"github.com/google/wire"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogmulti "github.com/samber/slog-multi"
	realotelslog "go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/gosundheit"
)

var TelemetryWithoutServer = wire.NewSet(
	NewTelemetryResource,

	NewLoggerProvider,
	wire.Bind(new(log.LoggerProvider), new(*sdklog.LoggerProvider)),
	NewLogger,

	NewMeterProvider,
	wire.Bind(new(metric.MeterProvider), new(*sdkmetric.MeterProvider)),
	NewMeter,
	NewTracerProvider,
	wire.Bind(new(trace.TracerProvider), new(*sdktrace.TracerProvider)),
	NewTracer,

	NewRuntimeMetricsCollector,
)

var Telemetry = wire.NewSet(
	NewTelemetryResource,

	NewLoggerProvider,
	wire.Bind(new(log.LoggerProvider), new(*sdklog.LoggerProvider)),
	NewLogger,

	NewMeterProvider,
	wire.Bind(new(metric.MeterProvider), new(*sdkmetric.MeterProvider)),
	NewMeter,
	NewTracerProvider,
	wire.Bind(new(trace.TracerProvider), new(*sdktrace.TracerProvider)),
	NewTracer,

	NewHealthChecker,

	NewTelemetryHandler,
	NewTelemetryServer,

	NewRuntimeMetricsCollector,
)

// Set the default logger to JSON for messages emitted before the "real" logger is initialized.
//
// We use JSON as a best-effort to make the logs machine-readable.
func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
}

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
			semconv.DeploymentEnvironmentName(metadata.Environment),
		),
	)

	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	return res
}

func NewLoggerProvider(ctx context.Context, conf config.LogTelemetryConfig, res *resource.Resource) (*sdklog.LoggerProvider, func(), error) {
	loggerProvider, err := conf.NewLoggerProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry Trace provider: %w", err)
	}

	return loggerProvider, func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
		ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
		defer cancel()

		if err := loggerProvider.ForceFlush(ctx); err != nil {
			// no logger initialized at this point yet, so we are using the global logger
			slog.Error("flushing logger provider", slog.String("error", err.Error()))
		}

		if err := loggerProvider.Shutdown(ctx); err != nil {
			// no logger initialized at this point yet, so we are using the global logger
			slog.Error("shutting down logger provider", slog.String("error", err.Error()))
		}
	}, nil
}

func NewLogger(conf config.LogTelemetryConfig, res *resource.Resource, loggerProvider log.LoggerProvider, metadata Metadata, additionalMiddlewares []slogmulti.Middleware) *slog.Logger {
	baseMiddlewares := []slogmulti.Middleware{
		otelslog.ResourceMiddleware(res),
		otelslog.NewHandler,
	}

	baseMiddlewares = append(baseMiddlewares, additionalMiddlewares...)

	// Stdout logger
	stdoutLogger := slogmulti.
		Pipe(baseMiddlewares...).
		Handler(conf.NewHandler(os.Stdout))

	// OTel logger
	// It already has the resource middleware applied by the loggerProvider
	otelLogger := NewLevelHandler(
		realotelslog.NewHandler(metadata.OpenTelemetryName, realotelslog.WithLoggerProvider(loggerProvider)),
		conf.Level,
	)

	// Fanout logger to stdout and OTel logger
	out := slogmulti.Fanout(
		stdoutLogger,
		otelLogger,
	)

	// Enrich log records
	middlewares := slogmulti.Pipe(
		contextx.NewLogHandler,
	)

	return slog.New(middlewares.Handler(out))
}

func TelemetryLoggerNoAdditionalMiddlewares() []slogmulti.Middleware {
	return nil
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

func NewTelemetryHandler(
	metricsConf config.MetricsTelemetryConfig,
	healthChecker health.Health,
	runtimeMetricsCollector RuntimeMetricsCollector,
	logger *slog.Logger,
) TelemetryHandler {
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

	// Start runtime metrics collector
	{
		if err := runtimeMetricsCollector.Start(); err != nil {
			logger.Error("failed to start runtime metrics collector, continuing startup", slog.String("error", err.Error()))
		}
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

type TelemetryMiddlewareHook server.MiddlewareHook

func NewTelemetryRouterHook(meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) TelemetryMiddlewareHook {
	return func(m server.MiddlewareManager) {
		m.Use(func(h http.Handler) http.Handler {
			return otelhttp.NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					h.ServeHTTP(w, r)

					routePattern := chi.RouteContext(r.Context()).RoutePattern()

					span := trace.SpanFromContext(r.Context())
					span.SetName(routePattern)
					span.SetAttributes(semconv.URLPath(r.URL.String()), semconv.HTTPRoute(routePattern))

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

type RuntimeMetricsCollector struct {
	meterProvider metric.MeterProvider
	logger        *slog.Logger
	conf          config.TelemetryConfig
}

func (c RuntimeMetricsCollector) Start() error {
	err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
		runtime.WithMeterProvider(c.meterProvider),
	)
	if err != nil {
		c.logger.Error("failed to start runtime metrics", slog.String("error", err.Error()))
		return err
	}

	c.logger.Debug("Started collecting runtime metrics")
	return nil
}

func NewRuntimeMetricsCollector(
	meterProvider metric.MeterProvider,
	conf config.TelemetryConfig,
	logger *slog.Logger,
) (RuntimeMetricsCollector, error) {
	return RuntimeMetricsCollector{
		meterProvider: meterProvider,
		logger:        logger,
		conf:          conf,
	}, nil
}

// Compile-time check LevelHandler implements slog.Handler.
var _ slog.Handler = (*LevelHandler)(nil)

// NewLevelHandler returns a new LevelHandler.
func NewLevelHandler(handler slog.Handler, level slog.Leveler) *LevelHandler {
	return &LevelHandler{
		handler: handler,
		level:   level,
	}
}

// LevelHandler is a slog.Handler that filters log records based on the log level.
type LevelHandler struct {
	handler slog.Handler
	level   slog.Leveler
}

func (h *LevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// The higher the level, the more important or severe the event.
	return level >= h.level.Level() && h.handler.Enabled(ctx, level)
}

func (h *LevelHandler) WithGroup(name string) slog.Handler {
	return NewLevelHandler(h.handler.WithGroup(name), h.level)
}

func (h *LevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewLevelHandler(h.handler.WithAttrs(attrs), h.level)
}

func (h *LevelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}
