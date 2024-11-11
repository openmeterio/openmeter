//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator

	Metadata common.Metadata

	StreamingConnector streaming.Connector
	MeterRepository    meter.Repository
	EntClient          *db.Client
	TelemetryServer    common.TelemetryServer
	BrokerOptions      watermillkafka.BrokerOptions
	MessagePublisher   message.Publisher
	EventPublisher     eventbus.Publisher

	Logger *slog.Logger
	Meter  metric.Meter
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		metadata,
		common.Config,
		common.Framework,
		common.Telemetry,
		common.NewDefaultTextMapPropagator,
		common.Database,
		common.ClickHouse,
		common.KafkaTopic,
		common.NotificationServiceProvisionTopics,
		common.Watermill,
		common.OpenMeter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "notification-worker")
}
