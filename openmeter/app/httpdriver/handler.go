package httpdriver

import (
	"context"
	"errors"
	"fmt"
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
	service app.Service

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
) (Handler, error) {
	var errs []error
	if logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if appService == nil {
		errs = append(errs, errors.New("app service is required"))
	}
	if appStripeService == nil {
		errs = append(errs, errors.New("app stripe service is required"))
	}
	if billingService == nil {
		errs = append(errs, errors.New("billing service is required"))
	}
	if customerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid app handler config: %w", err)
	}

	return &handler{
		service:          appService,
		namespaceDecoder: namespaceDecoder,
		stripeAppService: appStripeService,
		billingService:   billingService,
		customerService:  customerService,
		options:          options,
	}, nil
}
