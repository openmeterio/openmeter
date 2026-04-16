package appstripe

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Service interface {
	AppFactoryService
	StripeAppService
	CustomerService
	BillingService
}

// AppFactoryService contains methods to interface with app subsystem
type AppFactoryService interface {
	// App Factory methods
	NewApp(ctx context.Context, appBase app.AppBase) (app.App, error)
	InstallAppWithAPIKey(ctx context.Context, input app.AppFactoryInstallAppWithAPIKeyInput) (app.App, error)
	UninstallApp(ctx context.Context, input app.UninstallAppInput) error
}

// StripeAppService contains methods for managing stripe app
type StripeAppService interface {
	UpdateAPIKey(ctx context.Context, input UpdateAPIKeyInput) error
	GetStripeAppData(ctx context.Context, input GetStripeAppDataInput) (AppData, error)
	GetWebhookSecret(ctx context.Context, input GetWebhookSecretInput) (GetWebhookSecretOutput, error)
}

// CustomerService contains methods for managing customer data
type CustomerService interface {
	GetStripeCustomerData(ctx context.Context, input GetStripeCustomerDataInput) (CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input DeleteStripeCustomerDataInput) error
	HandleSetupIntentSucceeded(ctx context.Context, input HandleSetupIntentSucceededInput) (HandleSetupIntentSucceededOutput, error)

	CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (CreateCheckoutSessionOutput, error)
	CreatePortalSession(ctx context.Context, input CreateStripePortalSessionInput) (StripePortalSession, error)
}

// BillingService contains methods for managing billing subsystem (invoices)
type BillingService interface {
	GetSupplierContact(ctx context.Context, input GetSupplierContactInput) (billing.SupplierContact, error)

	// Invoice webhook handlers
	HandleInvoiceStateTransition(ctx context.Context, input HandleInvoiceStateTransitionInput) error
	HandleInvoiceSentEvent(ctx context.Context, input HandleInvoiceSentEventInput) error
}
