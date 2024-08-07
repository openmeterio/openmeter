package entitlement

import (
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func NewEntitlementConnector(
	edb EntitlementRepo,
	fc productcatalog.FeatureConnector,
	meterRepo meter.Repository,
	metered SubTypeConnector,
	static SubTypeConnector,
	boolean SubTypeConnector,
	eventBus eventbus.Publisher,
) EntitlementConnector {
	return entitlement.NewEntitlementConnector(edb, fc, meterRepo, metered, static, boolean, eventBus)
}
