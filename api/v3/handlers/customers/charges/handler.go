package charges

import (
	"context"

	"github.com/openmeterio/openmeter/api/v3/handlers/billingerrors"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCustomerCharges() ListCustomerChargesHandler
	CreateCustomerCharge() CreateCustomerChargesHandler
}

type handler struct {
	resolveNamespace  func(ctx context.Context) (string, error)
	service           billingcharges.Service
	unitConfigEnabled bool
	options           []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service billingcharges.Service,
	unitConfigEnabled bool,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(billingerrors.ErrorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace:  resolveNamespace,
		service:           service,
		unitConfigEnabled: unitConfigEnabled,
		options:           sharedOptions,
	}
}
