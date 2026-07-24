package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
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
	GetCustomerEntitlementValueV2() GetCustomerEntitlementValueV2Handler
	GetCustomerAccess() GetCustomerAccessHandler
	GetCustomerAccessV2() GetCustomerAccessV2Handler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service             customer.Service
	entitlementService  entitlement.Service
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
	subscriptionService subscription.Service,
	entitlementService entitlement.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if service == nil {
		errs = append(errs, errors.New("customer service is required"))
	}
	if subscriptionService == nil {
		errs = append(errs, errors.New("subscription service is required"))
	}
	if entitlementService == nil {
		errs = append(errs, errors.New("entitlement service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid customer handler config: %w", err)
	}

	return &handler{
		service:             service,
		subscriptionService: subscriptionService,
		namespaceDecoder:    namespaceDecoder,
		entitlementService:  entitlementService,
		options:             options,
	}, nil
}
