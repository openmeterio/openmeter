package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
	taxCodeResolver  productcatalog.TransactionalTaxCodeResolver
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
	chargeService billingcharges.ChargeService,
	credits config.CreditsConfiguration,
	featureGate *featuregate.FeatureGateChecker,
	taxCodeResolver productcatalog.TransactionalTaxCodeResolver,
	options ...httptransport.HandlerOption,
) Handler {
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
		taxCodeResolver:  taxCodeResolver,
	}
}

// ResolveWorkflowTaxCodes looks up, in one batch, the Tax Codes referenced by the workflows'
// default tax configs, so v1 responses can show their stripe.code.
func ResolveWorkflowTaxCodes(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace string, workflows ...billing.WorkflowConfig) (map[string]*taxcode.TaxCode, error) {
	ids := lo.Uniq(lo.FilterMap(workflows, func(wf billing.WorkflowConfig, _ int) (string, bool) {
		id := wf.Invoicing.DefaultTaxCodeID()
		return id, id != ""
	}))

	if len(ids) == 0 {
		return nil, nil
	}

	if resolver == nil {
		return nil, errors.New("tax code resolver is required to render default tax configs")
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
