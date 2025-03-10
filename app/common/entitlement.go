package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Entitlement = wire.NewSet(
	NewEntitlementRegistry,
)

func NewEntitlementRegistry(
	logger *slog.Logger,
	db *entdb.Client,
	entitlementConfig config.EntitlementsConfiguration,
	streamingConnector streaming.Connector,
	meterService meter.Service,
	eventPublisher eventbus.Publisher,
) *registry.Entitlement {
	return registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:            db,
		StreamingConnector:        streamingConnector,
		MeterService:              meterService,
		Logger:                    logger,
		Publisher:                 eventPublisher,
		EntitlementsConfiguration: entitlementConfig,
	})
}
