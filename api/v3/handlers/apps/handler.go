package apps

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListAppCatalogItems() ListAppCatalogItemsHandler
	GetAppCatalogItem() GetAppCatalogItemHandler
	GetAppCatalogItemOauth2InstallUrl() GetAppCatalogItemOauth2InstallUrlHandler
	SubmitCustomInvoicingDraftSynchronized() SubmitCustomInvoicingDraftSynchronizedHandler
	SubmitCustomInvoicingIssuingSynchronized() SubmitCustomInvoicingIssuingSynchronizedHandler
	UpdateCustomInvoicingPaymentStatus() UpdateCustomInvoicingPaymentStatusHandler
	CreateStripeCheckoutSession() CreateStripeCheckoutSessionHandler
	HandleStripeWebhook() HandleStripeWebhookHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          app.Service
	billingService   billing.Service
	customerService  customer.Service
	stripeService    appstripe.Service
	syncService      appcustominvoicing.SyncService
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	appService app.Service,
	billingService billing.Service,
	customerService customer.Service,
	stripeService appstripe.Service,
	syncService appcustominvoicing.SyncService,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          appService,
		billingService:   billingService,
		customerService:  customerService,
		stripeService:    stripeService,
		syncService:      syncService,
		options:          options,
	}
}
