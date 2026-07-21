package charges

import (
	"context"

	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCustomerCharges() ListCustomerChargesHandler
	CreateCustomerCharge() CreateCustomerChargesHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          billingcharges.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service billingcharges.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
