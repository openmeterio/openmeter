package appstripe

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
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
	UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error
	GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error)
	GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error)
}

// CustomerService contains methods for managing customer data
type CustomerService interface {
	GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
	HandleSetupIntentSucceeded(ctx context.Context, input appstripeentity.HandleSetupIntentSucceededInput) (appstripeentity.HandleSetupIntentSucceededOutput, error)

	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error)
	CreatePortalSession(ctx context.Context, input appstripeentity.CreateStripePortalSessionInput) (appstripeentity.StripePortalSession, error)
}

// BillingService contains methods for managing billing subsystem (invoices)
type BillingService interface {
	GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error)

	// Invoice webhook handlers
	HandleInvoiceStateTransition(ctx context.Context, input appstripeentity.HandleInvoiceStateTransitionInput) error
	HandleInvoiceSentEvent(ctx context.Context, input appstripeentity.HandleInvoiceSentEventInput) error
}
