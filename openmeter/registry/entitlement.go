package registry

import (
	"github.com/openmeterio/openmeter/internal/registry"
	registrybuilder "github.com/openmeterio/openmeter/internal/registry/builder"
)

type (
	Entitlement        = registry.Entitlement
	EntitlementOptions = registry.EntitlementOptions
)

func GetEntitlementRegistry(opts EntitlementOptions) *Entitlement {
	return registrybuilder.GetEntitlementRegistry(opts)
}
