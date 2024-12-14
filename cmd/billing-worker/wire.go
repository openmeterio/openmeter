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
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator
	common.Runner

	App                   app.Service
	AppStripe             appstripe.Service
	AppSandboxProvisioner common.AppSandboxProvisioner
	Logger                *slog.Logger
	Meter                 meter.Repository
	Streaming             streaming.Connector
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		wire.FieldsOf(new(config.BillingWorkerConfiguration), "ConsumerConfiguration"),
		wire.FieldsOf(new(config.BillingConfiguration), "Worker"),

		metadata,
		common.BillingWorker,
		common.ClickHouse,
		common.Config,
		common.Database,
		common.Framework,
		common.KafkaTopic,
		common.KafkaNamespaceResolver,
		common.MeterInMemory,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.Streaming,
		common.Telemetry,
		common.Watermill,
		common.WatermillRouter,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "billing-worker")
}
