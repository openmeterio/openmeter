//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator
	common.Runner

	App                   app.Service
	AppStripe             appstripe.Service
	AppSandboxProvisioner common.AppSandboxProvisioner
	Logger                *slog.Logger
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
		common.BillingWorker,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "billing-worker")
}
