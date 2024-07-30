package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	httpdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	EntitlementHandler        = httpdriver.EntitlementHandler
	MeteredEntitlementHandler = httpdriver.MeteredEntitlementHandler
)

type (
	CreateEntitlementHandler            = httpdriver.CreateEntitlementHandler
	CreateGrantHandler                  = httpdriver.CreateGrantHandler
	GetEntitlementBalanceHistoryHandler = httpdriver.GetEntitlementBalanceHistoryHandler
	GetEntitlementValueHandler          = httpdriver.GetEntitlementValueHandler
	GetEntitlementsOfSubjectHandler     = httpdriver.GetEntitlementsOfSubjectHandler
	ListEntitlementGrantsHandler        = httpdriver.ListEntitlementGrantsHandler
	ResetEntitlementUsageHandler        = httpdriver.ResetEntitlementUsageHandler
	ListEntitlementsHandler             = httpdriver.ListEntitlementsHandler
	GetEntitlementHandler               = httpdriver.GetEntitlementHandler
	GetEntitlementByIdHandler           = httpdriver.GetEntitlementByIdHandler
	DeleteEntitlementHandler            = httpdriver.DeleteEntitlementHandler
)

func NewEntitlementHandler(
	connector entitlement.EntitlementConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return httpdriver.NewEntitlementHandler(connector, namespaceDecoder, options...)
}

func NewMeteredEntitlementHandler(
	entitlementConnector entitlement.EntitlementConnector,
	meteredEntitlementConnector meteredentitlement.Connector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) MeteredEntitlementHandler {
	return httpdriver.NewMeteredEntitlementHandler(entitlementConnector, meteredEntitlementConnector, namespaceDecoder, options...)
}

func MapEntitlementValueToAPI(entitlementValue entitlement.EntitlementValue) (api.EntitlementValue, error) {
	return httpdriver.MapEntitlementValueToAPI(entitlementValue)
}
