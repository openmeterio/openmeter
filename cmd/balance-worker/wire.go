//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator
	common.Runner

	Logger *slog.Logger
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		metadata,

		wire.FieldsOf(new(config.Configuration), "BalanceWorker"),
		wire.FieldsOf(new(config.BalanceWorkerConfiguration), "ConsumerConfiguration"),

		common.BalanceWorker,
		common.BalanceWorkerAdapter,
		common.ClickHouse,
		common.Config,
		common.Customer,
		common.Database,
		common.Feature,
		common.NewLLMCostService,
		common.Framework,
		common.Meter,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.NewKafkaTopicProvisioner,
		common.Notification,
		common.ProgressManager,
		common.Streaming,
		common.Subject,
		common.NewSvixAPIClient,
		common.Telemetry,
		common.TelemetryLoggerNoAdditionalMiddlewares,
		common.Watermill,
		common.WatermillRouter,
		common.Lockr,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "balance-worker")
}
