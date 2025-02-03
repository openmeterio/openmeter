package appstripe

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
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
	NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error)
	InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error)
	UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error
}

// StripeAppService contains methods for managing stripe app
type StripeAppService interface {
	UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error
	GetMaskedSecretAPIKey(secretAPIKeyID secretentity.SecretID) (string, error)
	GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error)
	GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error)
}

// CustomerService contains methods for managing customer data
type CustomerService interface {
	GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
	SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error)

	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error)
}

// BillingService contains methods for managing billing subsystem (invoices)
type BillingService interface {
	GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error)

	// Invoice webhook handlers
	HandleInvoiceStateTransition(ctx context.Context, input appstripeentity.HandleInvoiceStateTransitionInput) error
	HandleInvoiceSentEvent(ctx context.Context, input appstripeentity.HandleInvoiceSentEventInput) error
}
