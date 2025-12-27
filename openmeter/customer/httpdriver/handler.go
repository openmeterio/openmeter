package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CustomerHandler
}

type CustomerHandler interface {
	ListCustomers() ListCustomersHandler
	CreateCustomer() CreateCustomerHandler
	DeleteCustomer() DeleteCustomerHandler
	GetCustomer() GetCustomerHandler
	UpdateCustomer() UpdateCustomerHandler
	GetCustomerEntitlementValue() GetCustomerEntitlementValueHandler
	GetCustomerAccess() GetCustomerAccessHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service             customer.Service
	entitlementService  entitlement.Service
	planService         plan.Service
	subscriptionService subscription.Service
	namespaceDecoder    namespacedriver.NamespaceDecoder
	options             []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service customer.Service,
	entitlementService entitlement.Service,
	planService plan.Service,
	subscriptionService subscription.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:             service,
		entitlementService:  entitlementService,
		planService:         planService,
		subscriptionService: subscriptionService,
		namespaceDecoder:    namespaceDecoder,
		options:             options,
	}
}
