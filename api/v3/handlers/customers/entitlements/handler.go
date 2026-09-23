package customersentitlements

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateCustomerEntitlement() CreateCustomerEntitlementHandler
	GetCustomerEntitlementHistory() GetCustomerEntitlementHistoryHandler
	GetCustomerEntitlement() GetCustomerEntitlementHandler
	ListCustomerEntitlements() ListCustomerEntitlementsHandler
	ResetCustomerEntitlementUsage() ResetCustomerEntitlementUsageHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          entitlement.CustomerEntitlementAPIService
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service entitlement.CustomerEntitlementAPIService,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(errorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          sharedOptions,
	}
}
