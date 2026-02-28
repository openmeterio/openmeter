//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator
	common.Runner

	AppRegistry      common.AppRegistry
	Logger           *slog.Logger
	Meter            meter.Service
	NamespaceManager *namespace.Manager
	Streaming        streaming.Connector
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		wire.FieldsOf(new(config.BillingWorkerConfiguration), "ConsumerConfiguration"),
		wire.FieldsOf(new(config.BillingConfiguration), "Worker"),

		metadata,
		common.BillingWorker,
		common.NewLLMCostService,
		common.ClickHouse,
		common.Config,
		common.Database,
		common.Framework,
		common.Meter,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.NewKafkaTopicProvisioner,
		common.ProgressManager,
		common.Streaming,
		common.Telemetry,
		common.TelemetryLoggerNoAdditionalMiddlewares,
		common.Watermill,
		common.WatermillRouter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "billing-worker")
}
