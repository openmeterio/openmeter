package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/governance"
	governanceservice "github.com/openmeterio/openmeter/openmeter/governance/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

var Governance = wire.NewSet(
	NewGovernanceService,
)

func NewGovernanceService(
	customerService customer.Service,
	entitlementRegistry *registry.Entitlement,
) (governance.Service, error) {
	return governanceservice.New(governanceservice.Config{
		CustomerService:    customerService,
		EntitlementService: entitlementRegistry.Entitlement,
		FeatureConnector:   entitlementRegistry.Feature,
	})
}
