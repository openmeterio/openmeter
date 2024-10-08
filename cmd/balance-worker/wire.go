//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Application struct {
	StreamingConnector streaming.Connector
	MeterRepository    meter.Repository
	EntClient          *db.Client
	TelemetryServer    app.TelemetryServer
	// EventPublisher     eventbus.Publisher

	Meter metric.Meter

	// TODO: move to global setter
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		app.Config,
		NewOtelResource,
		app.Telemetry,
		NewMeter,
		app.Database,
		app.ClickHouse,
		// app.Kafka,
		// app.Watermill,
		app.OpenMeter,
		// wire.Value(app.WatermillClientID(otelName)),
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	wire.Build(app.Config, NewOtelResource, app.Logger)

	return new(slog.Logger)
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
