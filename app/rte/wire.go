//go:build wireinject

package rte

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

	AppRegistry             common.AppRegistry
	RTE                     rte.Service
	Logger                  *slog.Logger
	RuntimeMetricsCollector common.RuntimeMetricsCollector
}

// Initialize builds the worker after the host has enabled RTE and validated its configuration.
func Initialize(ctx context.Context, conf config.Configuration, metadata common.Metadata) (Application, func(), error) {
	wire.Build(
		wire.FieldsOf(new(config.RTEConfiguration), "ConsumerConfiguration"),
		common.RTE,
		common.NewCustomerBalanceService,
		common.NewCreditVoidService,
		common.App,
		common.Secret,
		common.Billing,
		common.ClickHouse,
		common.Config,
		common.Currency,
		common.Customer,
		common.Database,
		common.NewEntitlementRegistry,
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
		common.WatermillRTE,
		common.NewKafkaTopicProvisioner,
		common.WatermillRouter,
		common.FeatureGateChecker,
		wire.Struct(new(Application), "*"),
	)

	return Application{}, nil, nil
}
