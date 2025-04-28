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
		wire.FieldsOf(new(config.BalanceWorkerConfiguration), "Estimator"),

		common.BalanceWorker,
		common.BalanceWorkerAdapter,
		common.ClickHouse,
		common.Config,
		common.Database,
		common.Feature,
		common.Framework,
		common.KafkaTopic,
		common.Meter,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.NewNotificationBalanceThresholdEventHandler,
		common.Notification,
		common.ProgressManager,
		common.Streaming,
		common.Telemetry,
		common.Watermill,
		common.WatermillRouter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "balance-worker")
}
