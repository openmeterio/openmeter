package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AppStripeHandler
}

type AppStripeHandler interface {
	AppStripeWebhook() AppStripeWebhookHandler
	UpdateStripeAPIKey() UpdateStripeAPIKeyHandler
	CreateAppStripeCheckoutSession() CreateAppStripeCheckoutSessionHandler

	// Customer Stripe Data handlers
	GetCustomerStripeAppData() GetCustomerStripeAppDataHandler
	UpsertCustomerStripeAppData() UpsertCustomerStripeAppDataHandler

	// Customer Stripe Portal handlers
	CreateStripeCustomerPortalSession() CreateStripeCustomerPortalSessionHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          appstripe.Service
	billingService   billing.Service
	customerService  customer.Service
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service appstripe.Service,
	billingService billing.Service,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if service == nil {
		errs = append(errs, errors.New("app stripe service is required"))
	}
	if billingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}
	if customerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid app stripe handler config: %w", err)
	}

	return &handler{
		service:          service,
		billingService:   billingService,
		customerService:  customerService,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}, nil
}
