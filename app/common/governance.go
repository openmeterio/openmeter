package common

import (
	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/governance"
	governanceservice "github.com/openmeterio/openmeter/openmeter/governance/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

var Governance = wire.NewSet(
	NewGovernanceService,
)

func NewGovernanceService(
	customer customer.Service,
	entitlementRegistry *registry.Entitlement,
	tracer trace.Tracer,
) (governance.Service, error) {
	return governanceservice.New(governanceservice.Config{
		Customer:    customer,
		Entitlement: entitlementRegistry.Entitlement,
		Feature:     entitlementRegistry.Feature,
		Tracer:      tracer,
	})
}
