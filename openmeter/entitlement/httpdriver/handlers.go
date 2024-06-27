package httpdriver

import (
	"github.com/openmeterio/openmeter/internal/entitlement/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type EntitlementHandler = httpdriver.EntitlementHandler
type MeteredEntitlementHandler = httpdriver.MeteredEntitlementHandler

type CreateEntitlementHandler = httpdriver.CreateEntitlementHandler
type CreateGrantHandler = httpdriver.CreateGrantHandler
type GetEntitlementBalanceHistoryHandler = httpdriver.GetEntitlementBalanceHistoryHandler
type GetEntitlementValueHandler = httpdriver.GetEntitlementValueHandler
type GetEntitlementsOfSubjectHandler = httpdriver.GetEntitlementsOfSubjectHandler
type DeleteEntitlementHandlerRequest = httpdriver.DeleteEntitlementHandlerRequest
type DeleteEntitlementHandlerResponse = httpdriver.DeleteEntitlementHandlerResponse
type ListEntitlementGrantsHandler = httpdriver.ListEntitlementGrantsHandler
type ResetEntitlementUsageHandler = httpdriver.ResetEntitlementUsageHandler
type ListEntitlementsHandler = httpdriver.ListEntitlementsHandler
type GetEntitlementHandler = httpdriver.GetEntitlementHandler
type DeleteEntitlementHandler = httpdriver.DeleteEntitlementHandler

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
