package httpdriver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ProfileHandler
	InvoiceLineHandler
	InvoiceHandler
	InvoiceCostHandler
	CustomerOverrideHandler
}

type ProfileHandler interface {
	CreateProfile() CreateProfileHandler
	GetProfile() GetProfileHandler
	DeleteProfile() DeleteProfileHandler
	UpdateProfile() UpdateProfileHandler
	ListProfiles() ListProfilesHandler
}

type InvoiceLineHandler interface {
	CreatePendingLine() CreatePendingLineHandler
}

type InvoiceHandler interface {
	ListInvoices() ListInvoicesHandler
	InvoicePendingLinesAction() InvoicePendingLinesActionHandler
	DeleteInvoice() DeleteInvoiceHandler
	GetInvoice() GetInvoiceHandler
	UpdateInvoice() UpdateInvoiceHandler
	ProgressInvoice(ProgressAction) ProgressInvoiceHandler
	SimulateInvoice() SimulateInvoiceHandler
}

type InvoiceCostHandler interface {
	GetInvoiceLineCost() GetInvoiceLineCostHandler
}

type CustomerOverrideHandler interface {
	ListCustomerOverrides() ListCustomerOverridesHandler
	UpsertCustomerOverride() UpsertCustomerOverrideHandler
	GetCustomerOverride() GetCustomerOverrideHandler
	DeleteCustomerOverride() DeleteCustomerOverrideHandler
}

type handler struct {
	service          billing.Service
	appService       app.Service
	streamingService streaming.Connector
	meterService     meter.Service
	featureService   feature.FeatureConnector
	namespaceDecoder namespacedriver.NamespaceDecoder
	featureSwitches  config.BillingFeatureSwitchesConfiguration
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
	featureSwitches config.BillingFeatureSwitchesConfiguration,
	service billing.Service,
	appService app.Service,
	stripeAppService appstripe.Service,
	streamingService streaming.Connector,
	featureService feature.FeatureConnector,
	meterService meter.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		appService:       appService,
		namespaceDecoder: namespaceDecoder,
		options:          options,
		featureSwitches:  featureSwitches,
		streamingService: streamingService,
		meterService:     meterService,
		featureService:   featureService,
	}
}
