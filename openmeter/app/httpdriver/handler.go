package httpdriver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AppHandler
}

type AppHandler interface {
	// App handlers
	ListApps() ListAppsHandler
	GetApp() GetAppHandler
	UninstallApp() UninstallAppHandler
	UpdateApp() UpdateAppHandler

	// Customer Data handlers
	ListCustomerData() ListCustomerDataHandler
	UpsertCustomerData() UpsertCustomerDataHandler
	DeleteCustomerData() DeleteCustomerDataHandler

	// Marketplace handlers
	ListMarketplaceListings() ListMarketplaceListingsHandler
	GetMarketplaceListing() GetMarketplaceListingHandler
	MarketplaceAppAPIKeyInstall() MarketplaceAppAPIKeyInstallHandler
	MarketplaceAppInstall() MarketplaceAppInstallHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	appMapper *AppMapper
	service   app.Service

	stripeAppService stripeapp.Service
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
	logger *slog.Logger,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	appService app.Service,
	appStripeService stripeapp.Service,
	billingService billing.Service,
	customerService customer.Service,

	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		appMapper:        NewAppMapper(logger, appStripeService),
		service:          appService,
		namespaceDecoder: namespaceDecoder,
		stripeAppService: appStripeService,
		billingService:   billingService,
		customerService:  customerService,
		options:          options,
	}
}
