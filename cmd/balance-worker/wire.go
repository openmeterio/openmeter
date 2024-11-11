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
		common.Config,
		common.Framework,
		common.Telemetry,
		common.NewDefaultTextMapPropagator,
		common.Database,
		common.ClickHouse,
		common.KafkaTopic,
		common.Watermill,
		common.WatermillRouter,
		common.OpenMeter,
		common.BalanceWorkerAdapter,
		common.BalanceWorker,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "balance-worker")
}
