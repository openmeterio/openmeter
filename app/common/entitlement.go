package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	entitlementvalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

var Entitlement = wire.NewSet(NewEntitlementRegistryWithShadow)

func NewEntitlementRegistry(
	logger *slog.Logger,
	db *entdb.Client,
	tracer trace.Tracer,
	entitlementConfig config.EntitlementsConfiguration,
	streamingConnector streaming.Connector,
	meterService meter.Service,
	eventPublisher eventbus.Publisher,
	locker *lockr.Locker,
	customerService customer.Service,
) (*registry.Entitlement, error) {
	return NewEntitlementRegistryWithShadow(logger, db, tracer, entitlementConfig, streamingConnector, meterService, eventPublisher, locker, customerService, nil)
}

func NewEntitlementRegistryWithShadow(
	logger *slog.Logger,
	db *entdb.Client,
	tracer trace.Tracer,
	entitlementConfig config.EntitlementsConfiguration,
	streamingConnector streaming.Connector,
	meterService meter.Service,
	eventPublisher eventbus.Publisher,
	locker *lockr.Locker,
	customerService customer.Service,
	recorder *shadow.Recorder,
) (*registry.Entitlement, error) {
	opts := registrybuilder.EntitlementOptions{
		DatabaseClient:            db,
		StreamingConnector:        streamingConnector,
		MeterService:              meterService,
		CustomerService:           customerService,
		Logger:                    logger,
		Publisher:                 eventPublisher,
		EntitlementsConfiguration: entitlementConfig,
		Tracer:                    tracer,
		Locker:                    locker,
	}
	if recorder != nil {
		opts.WrapMeteredConnector = func(connector meteredentitlement.Connector) meteredentitlement.Connector {
			return shadow.ObserveMeteredValues(connector, recorder)
		}
	}

	entRegistry := registrybuilder.GetEntitlementRegistry(opts)

	// Create and register the entitlement validator
	validator, err := entitlementvalidator.NewValidator(entRegistry.EntitlementRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement validator: %w", err)
	}

	customerService.RegisterRequestValidator(validator)

	return entRegistry, nil
}
