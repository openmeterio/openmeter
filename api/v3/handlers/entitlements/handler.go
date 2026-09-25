package entitlements

import (
	"context"

	customersentitlements "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlements"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	GetEntitlement() GetEntitlementHandler
	ListEntitlements() ListEntitlementsHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          entitlement.EntitlementAPIService
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service entitlement.EntitlementAPIService,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(customersentitlements.ErrorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          sharedOptions,
	}
}
