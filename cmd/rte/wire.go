//go:build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/rte"
)

type Application struct {
	common.GlobalInitializer
	common.Migrator
	common.Runner

	RTE                     rte.Service
	Logger                  *slog.Logger
	RuntimeMetricsCollector common.RuntimeMetricsCollector
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		metadata,
		wire.FieldsOf(new(config.RTEConfiguration), "ConsumerConfiguration"),
		common.RTE,
		common.NewCustomerBalanceService,
		common.NewCreditVoidService,
		common.App,
		common.Billing,
		common.ClickHouse,
		common.Config,
		common.Currency,
		common.Customer,
		common.Database,
		common.Entitlement,
		common.Framework,
		common.FFX,
		common.Meter,
		common.Namespace,
		common.NewDefaultTextMapPropagator,
		common.ProductCatalog,
		common.ProgressManager,
		common.Streaming,
		common.TaxCode,
		common.Subscription,
		common.LedgerStack,
		common.Lockr,
		common.Telemetry,
		common.TelemetryLoggerNoAdditionalMiddlewares,
		common.Watermill,
		common.NewKafkaTopicProvisioner,
		common.WatermillRouter,
		common.FeatureGateChecker,
		wire.Struct(new(Application), "*"),
	)

	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "rte")
}
