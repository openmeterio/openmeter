package entitlementaccess

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	QueryEntitlementAccess() QueryEntitlementAccessHandler
}

type handler struct {
	resolveNamespace         func(ctx context.Context) (string, error)
	entitlementAccessService entitlementaccess.Service
	options                  []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	entitlementAccessService entitlementaccess.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:         resolveNamespace,
		entitlementAccessService: entitlementAccessService,
		options:                  options,
	}
}
