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
	wire.FieldsOf(new(config.Configuration), "Entitlements"),

	NewEntitlementRegistry,
)

func NewEntitlementRegistry(
	logger *slog.Logger,
	db *entdb.Client,
	entitlementConfig config.EntitlementsConfiguration,
	streamingConnector streaming.Connector,
	meterRepository meter.Repository,
	eventPublisher eventbus.Publisher,
) *registry.Entitlement {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil
	}

	if !entitlementConfig.Enabled {
		return nil
	}

	return registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     db,
		StreamingConnector: streamingConnector,
		MeterRepository:    meterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})
}
