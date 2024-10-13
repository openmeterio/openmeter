//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type Application struct {
	app.GlobalInitializer

	Metadata app.Metadata

	StreamingConnector streaming.Connector
	MeterRepository    meter.Repository
	EntClient          *db.Client
	TelemetryServer    app.TelemetryServer
	BrokerOptions      watermillkafka.BrokerOptions
	MessagePublisher   message.Publisher
	EventPublisher     eventbus.Publisher

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
		provisionTopics,
		app.Watermill,
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

func provisionTopics(conf config.BalanceWorkerConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.DLQ.Topic,
			Partitions:    conf.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}
