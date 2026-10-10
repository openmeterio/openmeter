package grants

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListGrants() ListGrantsHandler
	VoidGrant() VoidGrantHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          entitlement.GrantAPIService
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service entitlement.GrantAPIService,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
