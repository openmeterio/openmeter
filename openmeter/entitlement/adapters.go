package entitlement

import (
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func NewEntitlementConnector(
	edb EntitlementRepo,
	fc productcatalog.FeatureConnector,
	metered SubTypeConnector,
	static SubTypeConnector,
	boolean SubTypeConnector,
) EntitlementConnector {
	return entitlement.NewEntitlementConnector(edb, fc, metered, static, boolean)
}
