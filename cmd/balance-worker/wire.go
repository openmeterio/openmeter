//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type Application struct {
	app.GlobalInitializer

	Metadata app.Metadata

	StreamingConnector streaming.Connector
	MeterRepository    meter.Repository
	EntClient          *db.Client
	TelemetryServer    app.TelemetryServer
	// EventPublisher     eventbus.Publisher
	TopicProvisioner pkgkafka.TopicProvisioner

	Meter metric.Meter
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		metadata,
		app.Config,
		app.Framework,
		app.Telemetry,
		app.NewDefaultTextMapPropagator,
		app.Database,
		app.ClickHouse,
		app.KafkaTopic,
		// app.Watermill,
		app.OpenMeter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	wire.Build(metadata, app.Config, app.Logger)

	return new(slog.Logger)
}

func metadata(conf config.Configuration) app.Metadata {
	return app.Metadata{
		ServiceName:       "openmeter",
		Version:           version,
		Environment:       conf.Environment,
		OpenTelemetryName: "openmeter.io/balance-worker",
	}
}
