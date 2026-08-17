package common

import (
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
	entitlementaccessservice "github.com/openmeterio/openmeter/openmeter/entitlementaccess/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

var EntitlementAccess = wire.NewSet(
	NewEntitlementAccessService,
)

func NewEntitlementAccessService(
	customer customer.Service,
	entitlementRegistry *registry.Entitlement,
	tracer trace.Tracer,
	meter metric.Meter,
) (entitlementaccess.Service, error) {
	return entitlementaccessservice.New(entitlementaccessservice.Config{
		Customer:    customer,
		Entitlement: entitlementRegistry.Entitlement,
		Feature:     entitlementRegistry.Feature,
		Tracer:      tracer,
		Meter:       meter,
	})
}
