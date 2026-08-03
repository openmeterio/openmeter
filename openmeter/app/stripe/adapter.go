package appstripe

import (
	"context"

	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// WebhookSecretService resolves the signing secret used to authenticate incoming Stripe webhooks.
// It is separate from the app's general secret service so deployments can apply read-specific
// behavior, such as caching, to the high-volume webhook path.
type WebhookSecretService interface {
	GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error)
}

type Adapter interface {
	AppStripeAdapter

	entutils.TxCreator
}

type AppStripeAdapter interface {
	GetStripeClientFactory() client.StripeClientFactory
	GetStripeAppClientFactory() client.StripeAppClientFactory

	UpdateAPIKey(ctx context.Context, input UpdateAPIKeyAdapterInput) error
	CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (CreateCheckoutSessionOutput, error)
	GetWebhookSecret(ctx context.Context, input GetWebhookSecretInput) (GetWebhookSecretOutput, error)
	// App
	CreateStripeApp(ctx context.Context, input CreateAppStripeInput) (AppBase, error)
	GetStripeAppData(ctx context.Context, input GetStripeAppDataInput) (AppData, error)
	DeleteStripeAppData(ctx context.Context, input DeleteStripeAppDataInput) error
	// Billing
	GetSupplierContact(ctx context.Context, input GetSupplierContactInput) (billing.SupplierContact, error)
	GetStripeInvoice(ctx context.Context, input GetStripeInvoiceInput) (*stripe.Invoice, error)
	// Customer
	GetStripeCustomerData(ctx context.Context, input GetStripeCustomerDataInput) (CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input DeleteStripeCustomerDataInput) error
	SetCustomerDefaultPaymentMethod(ctx context.Context, input SetCustomerDefaultPaymentMethodInput) (SetCustomerDefaultPaymentMethodOutput, error)
	// Portal
	CreatePortalSession(ctx context.Context, input CreateStripePortalSessionInput) (StripePortalSession, error)
}
