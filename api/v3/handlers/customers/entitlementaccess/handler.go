package customersentitlement

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCustomerEntitlementAccess() ListCustomerEntitlementAccessHandler
}

type handler struct {
	resolveNamespace   func(ctx context.Context) (string, error)
	customerService    customer.Service
	entitlementService entitlement.Service
	options            []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	entitlementService entitlement.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:   resolveNamespace,
		customerService:    customerService,
		entitlementService: entitlementService,
		options:            options,
	}
}
