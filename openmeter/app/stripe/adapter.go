package appstripe

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	AppStripeAdapter

	entutils.TxCreator
}

type AppStripeAdapter interface {
	GetStripeClientFactory() client.StripeClientFactory
	GetStripeAppClientFactory() client.StripeAppClientFactory

	UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error
	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error)
	GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error)
	GetMaskedSecretAPIKey(secretAPIKeyID secretentity.SecretID) (string, error)
	// App
	CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.AppBase, error)
	GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error)
	DeleteStripeAppData(ctx context.Context, input appstripeentity.DeleteStripeAppDataInput) error
	// Billing
	GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error)
	// Customer
	GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
	SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error)
}
