//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-slog/otelslog"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

type Application struct {
	app.GlobalInitializer

	MeterRepository meter.Repository
	TelemetryServer app.TelemetryServer

	Meter metric.Meter

	// TODO: move to global setter
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		app.Config,
		app.Framework,
		NewOtelResource,
		app.Telemetry,
		NewMeter,
		NewTextMapPropagator,
		app.OpenMeter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	wire.Build(app.Config, NewOtelResource, app.Logger)

	return new(slog.Logger)
}

// TODO: use the primary logger
func NewLogger(conf config.Configuration, res *resource.Resource) *slog.Logger {
	logger := slog.New(otelslog.NewHandler(conf.Telemetry.Log.NewHandler(os.Stdout)))
	logger = otelslog.WithResource(logger, res)

	return logger
}

// TODO: consider moving this to a separate package
// TODO: make sure this doesn't generate any random IDs, because it is called twice
func NewOtelResource(conf config.Configuration) *resource.Resource {
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

// TODO: consider moving this to a separate package
func NewMeter(meterProvider metric.MeterProvider) metric.Meter {
	return meterProvider.Meter(otelName)
}

// TODO: consider moving this to a separate package
func NewTextMapPropagator() propagation.TextMapPropagator {
	return propagation.TraceContext{}
}
