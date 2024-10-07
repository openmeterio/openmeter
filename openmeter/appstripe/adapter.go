package appstripe

import (
	"context"

	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	AppStripeAdapter

	entutils.TxCreator
}

type AppStripeAdapter interface {
	CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error)
	GetStripeApp(ctx context.Context, input appstripeentity.GetAppInput) (appstripeentity.App, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error

	// Service methods
	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error)
	GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error)
	SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error)
}
