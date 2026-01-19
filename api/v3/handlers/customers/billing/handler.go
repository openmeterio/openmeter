package customersbilling

import (
	"context"

	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateStripeCheckoutSession() CreateStripeCheckoutSessionHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	billingService   billing.Service
	stripeService    appstripe.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	billingService billing.Service,
	stripeService appstripe.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		billingService:   billingService,
		stripeService:    stripeService,
		options:          options,
	}
}
