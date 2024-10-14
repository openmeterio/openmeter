package appservice

import (
	"context"

	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
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

func (s *Service) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerAppData, error) {
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
