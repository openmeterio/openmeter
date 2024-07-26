package entitlement

import (
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/publisher"
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
	publisher publisher.TopicPublisher,
) EntitlementConnector {
	return entitlement.NewEntitlementConnector(edb, fc, meterRepo, metered, static, boolean, publisher)
}
