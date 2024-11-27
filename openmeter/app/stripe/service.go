package appstripe

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
)

type Service interface {
	AppService
}

type AppService interface {
	// Marketplace
	NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error)
	InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error)
	UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error

	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error)
	GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error)
	// App
	GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error)
	// Customer
	GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
	SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error)
}
