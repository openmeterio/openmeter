package entitlement

import (
	"log/slog"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewEntitlementBalanceConnector(
	sc streaming.Connector,
	oc credit.OwnerConnector,
	bc credit.BalanceConnector,
	gc credit.GrantConnector,
) EntitlementBalanceConnector {
	return entitlement.NewEntitlementBalanceConnector(sc, oc, bc, gc)
}

func NewEntitlementConnector(
	ebc EntitlementBalanceConnector,
	edb EntitlementDBConnector,
	fc productcatalog.FeatureConnector,
) EntitlementConnector {
	return entitlement.NewEntitlementConnector(ebc, edb, fc)
}

func NewEntitlementGrantOwnerAdapter(
	fdb productcatalog.FeatureDBConnector,
	edb EntitlementDBConnector,
	urdb UsageResetDBConnector,
	mr meter.Repository,
	logger *slog.Logger,
) credit.OwnerConnector {
	return entitlement.NewEntitlementGrantOwnerAdapter(fdb, edb, urdb, mr, logger)
}
