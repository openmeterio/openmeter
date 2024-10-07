package httpdriver

import (
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AppStripeHandler
}

type AppStripeHandler interface {
	AppStripeWebhook() AppStripeWebhookHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          appstripe.Service
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service appstripe.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}
}
