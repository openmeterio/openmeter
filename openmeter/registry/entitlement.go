package registry

import "github.com/openmeterio/openmeter/internal/registry"

type (
	Entitlement        = registry.Entitlement
	EntitlementOptions = registry.EntitlementOptions
)

func GetEntitlementRegistry(opts EntitlementOptions) *Entitlement {
	return registry.GetEntitlementRegistry(opts)
}
