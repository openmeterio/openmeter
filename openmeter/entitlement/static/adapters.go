package staticentitlement

import staticentitlement "github.com/openmeterio/openmeter/internal/entitlement/static"

func NewStaticEntitlementConnector() Connector {
	return staticentitlement.NewStaticEntitlementConnector()
}
