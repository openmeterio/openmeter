package billingprofiles

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListBillingProfiles() ListBillingProfilesHandler
	CreateBillingProfile() CreateBillingProfileHandler
	GetBillingProfile() GetBillingProfileHandler
	UpdateBillingProfile() UpdateBillingProfileHandler
	DeleteBillingProfile() DeleteBillingProfileHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          billing.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service billing.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
