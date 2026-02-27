//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator

	BrokerOptions      watermillkafka.BrokerOptions
	EventPublisher     eventbus.Publisher
	EntClient          *db.Client
	FeatureConnector   feature.FeatureConnector
	Logger             *slog.Logger
	MessagePublisher   message.Publisher
	Meter              metric.Meter
	Tracer             trace.Tracer
	Metadata           common.Metadata
	MeterService       meter.Service
	Notification       notification.Service
	StreamingConnector streaming.Connector
	TelemetryServer    common.TelemetryServer
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		metadata,
		common.ClickHouse,
		common.Config,
		common.Database,
		common.Feature,
		common.NewLLMCostService,
		common.Framework,
		common.Meter,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.NewKafkaTopicProvisioner,
		common.Notification,
		common.NotificationServiceProvisionTopics,
		common.ProgressManager,
		common.Streaming,
		common.NewSvixAPIClient,
		common.Telemetry,
		common.TelemetryLoggerNoAdditionalMiddlewares,
		common.Watermill,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "notification-worker")
}
