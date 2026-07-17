package entitlementdriverv2

import (
	"errors"
	"fmt"

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
) (EntitlementHandler, error) {
	var errs []error
	if connector == nil {
		errs = append(errs, errors.New("entitlement connector is required"))
	}
	if balanceConnector == nil {
		errs = append(errs, errors.New("entitlement balance connector is required"))
	}
	if customerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid entitlement v2 handler config: %w", err)
	}

	return &entitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		balanceConnector: balanceConnector,
		customerService:  customerService,
	}, nil
}
