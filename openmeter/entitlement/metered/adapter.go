package meteredentitlement

import (
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewMeteredEntitlementConnector(
	streamingConnector streaming.Connector,
	ownerConnector credit.OwnerConnector,
	balanceConnector credit.BalanceConnector,
	grantConnector credit.GrantConnector,
	entitlementRepo entitlement.EntitlementRepo,
) Connector {
	return meteredentitlement.NewMeteredEntitlementConnector(
		streamingConnector,
		ownerConnector,
		balanceConnector,
		grantConnector,
		entitlementRepo,
	)
}
