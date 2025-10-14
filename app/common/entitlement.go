package common

import (
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	productcatalogpgadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

var Entitlement = wire.NewSet(
	NewEntitlementRegistry,
)

func NewEntitlementRegistry(
	logger *slog.Logger,
	db *entdb.Client,
	tracer trace.Tracer,
	entitlementConfig config.EntitlementsConfiguration,
	streamingConnector streaming.Connector,
	meterService meter.Service,
	eventPublisher eventbus.Publisher,
	locker *lockr.Locker,
	modelCostProvider *productcatalogpgadapter.ModelCostProvider,
) *registry.Entitlement {
	return registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:            db,
		StreamingConnector:        streamingConnector,
		MeterService:              meterService,
		Logger:                    logger,
		Publisher:                 eventPublisher,
		EntitlementsConfiguration: entitlementConfig,
		Tracer:                    tracer,
		Locker:                    locker,
		ModelCostProvider:         modelCostProvider,
	})
}
