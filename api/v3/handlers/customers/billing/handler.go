package customersbilling

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	GetCustomerBilling() GetCustomerBillingHandler
	UpdateCustomerBilling() UpdateCustomerBillingHandler
	UpdateCustomerBillingAppData() UpdateCustomerBillingAppDataHandler
	CreateCustomerStripeCheckoutSession() CreateCustomerStripeCheckoutSessionHandler
	CreateCustomerStripePortalSession() CreateCustomerStripePortalSessionHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	billingService   billing.Service
	customerService  customer.Service
	appService       app.Service
	stripeService    appstripe.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	billingService billing.Service,
	customerService customer.Service,
	stripeService appstripe.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		billingService:   billingService,
		customerService:  customerService,
		stripeService:    stripeService,
		options:          options,
	}
}
