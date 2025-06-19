package httpdriver

import (
	"context"
	"errors"
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
) Handler {
	return &handler{
		service:          service,
		billingService:   billingService,
		customerService:  customerService,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}
}
