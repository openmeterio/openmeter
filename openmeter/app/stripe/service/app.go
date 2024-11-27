package appservice

import (
	"context"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	return s.adapter.GetWebhookSecret(ctx, input)
}

func (s *Service) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	return s.adapter.CreateCheckoutSession(ctx, input)
}

func (s *Service) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	return s.adapter.GetStripeAppData(ctx, input)
}

func (s *Service) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error) {
	return s.adapter.GetStripeCustomerData(ctx, input)
}

func (s *Service) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	return s.adapter.UpsertStripeCustomerData(ctx, input)
}

func (s *Service) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	return s.adapter.DeleteStripeCustomerData(ctx, input)
}

func (s *Service) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	return s.adapter.SetCustomerDefaultPaymentMethod(ctx, input)
}

// newApp maps a stripe app to an app
func (s *Service) newApp(appBase appentitybase.AppBase, stripeApp appstripeentity.AppData) (appstripeentityapp.App, error) {
	app := appstripeentityapp.App{
		AppBase:             appBase,
		AppData:             stripeApp,
		StripeAppService:    s,
		SecretService:       s.secretService,
		StripeClientFactory: s.adapter.GetStripeClientFactory(),
	}

	if err := app.Validate(); err != nil {
		return appstripeentityapp.App{}, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}
