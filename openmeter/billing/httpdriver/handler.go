package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ProfileHandler
	InvoiceLineHandler
	InvoiceHandler
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

type CustomerOverrideHandler interface {
	ListCustomerOverrides() ListCustomerOverridesHandler
	UpsertCustomerOverride() UpsertCustomerOverrideHandler
	GetCustomerOverride() GetCustomerOverrideHandler
	DeleteCustomerOverride() DeleteCustomerOverrideHandler
}

type handler struct {
	service          billing.Service
	chargeService    billingcharges.ChargeService
	appService       app.Service
	logger           *slog.Logger
	namespaceDecoder namespacedriver.NamespaceDecoder
	featureSwitches  config.BillingFeatureSwitchesConfiguration
	credits          config.CreditsConfiguration
	featureGate      *featuregate.FeatureGateChecker
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
	chargeService billingcharges.ChargeService,
	credits config.CreditsConfiguration,
	featureGate *featuregate.FeatureGateChecker,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if service == nil {
		errs = append(errs, errors.New("billing service is required"))
	}
	if appService == nil {
		errs = append(errs, errors.New("app service is required"))
	}
	if chargeService == nil {
		errs = append(errs, errors.New("charge service is required"))
	}
	if featureGate == nil {
		errs = append(errs, errors.New("feature gate is required"))
	} else if err := featureGate.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature gate: %w", err))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid billing handler config: %w", err)
	}

	return &handler{
		service:          service,
		chargeService:    chargeService,
		appService:       appService,
		logger:           logger,
		namespaceDecoder: namespaceDecoder,
		options:          options,
		featureSwitches:  featureSwitches,
		credits:          credits,
		featureGate:      featureGate,
	}, nil
}
