package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

var Governance = wire.NewSet(
	NewGovernanceService,
)

func NewGovernanceService(
	customerService customer.Service,
	entitlementRegistry *registry.Entitlement,
) (governance.Service, error) {
	return governance.New(governance.Config{
		CustomerService:    customerService,
		EntitlementService: entitlementRegistry.Entitlement,
		FeatureConnector:   entitlementRegistry.Feature,
	})
}
