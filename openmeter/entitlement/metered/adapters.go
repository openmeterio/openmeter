package meteredentitlement

import (
	"log/slog"

	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewMeteredEntitlementConnector(
	streamingConnector streaming.Connector,
	ownerConnector credit.OwnerConnector,
	balanceConnector credit.BalanceConnector,
	grantConnector credit.GrantConnector,
	grantRepo credit.GrantRepo,
	entitlementRepo entitlement.EntitlementRepo,
	publisher eventbus.Publisher,
) Connector {
	return meteredentitlement.NewMeteredEntitlementConnector(
		streamingConnector,
		ownerConnector,
		balanceConnector,
		grantConnector,
		grantRepo,
		entitlementRepo,
		publisher,
	)
}

func NewEntitlementGrantOwnerAdapter(
	featureRepo productcatalog.FeatureRepo,
	entitlementRepo entitlement.EntitlementRepo,
	usageResetRepo UsageResetRepo,
	meterRepo meter.Repository,
	logger *slog.Logger,
) credit.OwnerConnector {
	return meteredentitlement.NewEntitlementGrantOwnerAdapter(
		featureRepo,
		entitlementRepo,
		usageResetRepo,
		meterRepo,
		logger,
	)
}
