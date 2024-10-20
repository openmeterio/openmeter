package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-slog/otelslog"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

const (
	defaultShutdownTimeout = 5 * time.Second
)

type Telemetry struct {
	Logger      *slog.Logger
	MetricMeter metric.Meter
	Shutdown    func()
}

func NewTelemetry(ctx context.Context, conf config.TelemetryConfig, env string, version string, otelName string) (*Telemetry, error) {
	extraResources, _ := resource.New(
		context.Background(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("openmeter"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(env),
		),
	)
	res, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	logger := slog.New(slogmulti.Pipe(
		otelslog.ResourceMiddleware(res),
		otelslog.NewHandler,
		contextx.NewLogHandler,
		operation.NewLogHandler,
	).Handler(conf.Log.NewHandler(os.Stdout)))

	slog.SetDefault(logger)

	// Initialize OTel Metrics
	otelMeterProvider, err := conf.Metrics.NewMeterProvider(ctx, res)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry Metrics provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		// Use dedicated context with timeout for shutdown as parent context might be canceled
		// by the time the execution reaches this stage.
	}()
	otel.SetMeterProvider(otelMeterProvider)
	metricMeter := otelMeterProvider.Meter(otelName)

	// Initialize OTel Tracer
	otelTracerProvider, err := conf.Trace.NewTracerProvider(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry Trace provider: %w", err)
	}

	otel.SetTracerProvider(otelTracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Telemetry{
		Logger:      logger,
		MetricMeter: metricMeter,
		Shutdown: func() {
			// Use dedicated context with timeout for shutdown as parent context might be canceled
			// by the time the execution reaches this stage.
			ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
			defer cancel()

			if err := otelMeterProvider.Shutdown(ctx); err != nil {
				logger.Error("shutting down meter provider", slog.String("error", err.Error()))
			}

			if err := otelTracerProvider.Shutdown(ctx); err != nil {
				logger.Error("shutting down tracer provider", slog.String("error", err.Error()))
			}
		},
	}, nil
}
