//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

type Application struct {
	app.GlobalInitializer

	StreamingConnector streaming.Connector
	MeterRepository    meter.Repository
	EntClient          *db.Client
	TelemetryServer    app.TelemetryServer
	KafkaProducer      *kafka.Producer
	KafkaMetrics       *kafkametrics.Metrics
	EventPublisher     eventbus.Publisher

	IngestCollector ingest.Collector

	NamespaceHandlers []namespace.Handler
	NamespaceManager  *namespace.Manager

	Meter metric.Meter

	RouterHook func(chi.Router)
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		metadata,
		app.Config,
		app.Framework,
		app.Telemetry,
		app.NewDefaultTextMapPropagator,
		app.NewTelemetryRouterHook,
		app.Database,
		app.ClickHouse,
		app.Kafka,
		app.ServerProvisionTopics,
		app.WatermillNoPublisher,
		app.NewServerPublisher,
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
		OpenTelemetryName: "openmeter.io/backend",
	}
}
