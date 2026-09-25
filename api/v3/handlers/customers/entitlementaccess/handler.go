package customersentitlement

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCustomerEntitlementAccess() ListCustomerEntitlementAccessHandler
	GetCustomerEntitlementAccess() GetCustomerEntitlementAccessHandler
	GetCustomerEntitlementValue() GetCustomerEntitlementValueHandler
	GetCustomerEntitlementValueByFeatureKey() GetCustomerEntitlementValueByFeatureKeyHandler
}

type handler struct {
	resolveNamespace   func(ctx context.Context) (string, error)
	entitlementService entitlement.Service
	options            []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	entitlementService entitlement.Service,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(errorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace:   resolveNamespace,
		entitlementService: entitlementService,
		options:            sharedOptions,
	}
}
