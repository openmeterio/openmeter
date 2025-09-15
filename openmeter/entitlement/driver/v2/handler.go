package entitlementdriverv2

import (
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// EntitlementHandler exposes V2 customer entitlement endpoints
type EntitlementHandler interface {
	CreateCustomerEntitlement() CreateCustomerEntitlementHandler
	ListCustomerEntitlements() ListCustomerEntitlementsHandler
	GetCustomerEntitlement() GetCustomerEntitlementHandler
	DeleteCustomerEntitlement() DeleteCustomerEntitlementHandler
	OverrideCustomerEntitlement() OverrideCustomerEntitlementHandler
	ListCustomerEntitlementGrants() ListCustomerEntitlementGrantsHandler
	CreateCustomerEntitlementGrant() CreateCustomerEntitlementGrantHandler
	GetCustomerEntitlementHistory() GetCustomerEntitlementHistoryHandler
	ResetCustomerEntitlementUsage() ResetCustomerEntitlementUsageHandler
	ListEntitlements() ListEntitlementsHandler
	GetEntitlement() GetEntitlementHandler
}

type entitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.Service
	balanceConnector meteredentitlement.Connector
	customerService  customer.Service
}

func NewEntitlementHandler(
	connector entitlement.Service,
	balanceConnector meteredentitlement.Connector,
	customerService customer.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return &entitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		balanceConnector: balanceConnector,
		customerService:  customerService,
	}
}
