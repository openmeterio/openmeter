package entitlementdriverv2

import (
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// CustomerEntitlementHandler exposes V2 customer entitlement endpoints
type CustomerEntitlementHandler interface {
	CreateCustomerEntitlement() CreateCustomerEntitlementHandler
	ListCustomerEntitlements() ListCustomerEntitlementsHandler
	GetCustomerEntitlement() GetCustomerEntitlementHandler
	DeleteCustomerEntitlement() DeleteCustomerEntitlementHandler
	OverrideCustomerEntitlement() OverrideCustomerEntitlementHandler
	ListCustomerEntitlementGrants() ListCustomerEntitlementGrantsHandler
	CreateCustomerEntitlementGrant() CreateCustomerEntitlementGrantHandler
	GetCustomerEntitlementHistory() GetCustomerEntitlementHistoryHandler
	ResetCustomerEntitlementUsage() ResetCustomerEntitlementUsageHandler
}

type customerEntitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.Connector
	balanceConnector meteredentitlement.Connector
	customerService  customer.Service
}

func NewCustomerEntitlementHandler(
	connector entitlement.Connector,
	balanceConnector meteredentitlement.Connector,
	customerService customer.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) CustomerEntitlementHandler {
	return &customerEntitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		balanceConnector: balanceConnector,
		customerService:  customerService,
	}
}
