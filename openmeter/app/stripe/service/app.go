package appservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ appstripe.Service = (*Service)(nil)

func (s *Service) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripeentity.GetWebhookSecretOutput, error) {
		return s.adapter.GetWebhookSecret(ctx, input)
	})
}

func (s *Service) UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyAdapterInput{
			UpdateAPIKeyInput: input,
			MaskedAPIKey:      s.generateMaskedSecretAPIKey(input.APIKey),
		})
	})
}

func (s *Service) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripeentity.CreateCheckoutSessionOutput, error) {
		return s.adapter.CreateCheckoutSession(ctx, input)
	})
}

func (s *Service) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripeentity.AppData, error) {
		return s.adapter.GetStripeAppData(ctx, input)
	})
}

func (s *Service) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripeentity.CustomerData, error) {
		return s.adapter.GetStripeCustomerData(ctx, input)
	})
}

func (s *Service) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.UpsertStripeCustomerData(ctx, input)
	})
}

func (s *Service) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteStripeCustomerData(ctx, input)
	})
}

func (s *Service) HandleSetupIntentSucceeded(ctx context.Context, input appstripeentity.HandleSetupIntentSucceededInput) (appstripeentity.HandleSetupIntentSucceededOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripeentity.HandleSetupIntentSucceededOutput, error) {
		def := appstripeentity.HandleSetupIntentSucceededOutput{}

		res, err := s.adapter.SetCustomerDefaultPaymentMethod(ctx, input.SetCustomerDefaultPaymentMethodInput)
		if err != nil {
			return def, fmt.Errorf("failed to set customer default payment method: %w", err)
		}

		handlingApp, err := s.appService.GetApp(ctx, input.AppID)
		if err != nil {
			return def, fmt.Errorf("failed to get app: %w", err)
		}

		event := app.CustomerPaymentSetupSucceededEvent{
			App:      handlingApp.GetAppBase(),
			Customer: res.CustomerID,
			Result: app.CustomerPaymentSetupResult{
				Metadata: lo.OmitByKeys(input.PaymentIntentMetadata, stripeclient.SetupIntentReservedMetadataKeys),
			},
		}

		if err := s.publisher.Publish(ctx, event); err != nil {
			return def, fmt.Errorf("failed to publish event: %w", err)
		}

		return appstripeentity.HandleSetupIntentSucceededOutput(res), nil
	})
}

func (s *Service) generateMaskedSecretAPIKey(secretAPIKey string) string {
	return fmt.Sprintf("%s***%s", secretAPIKey[:8], secretAPIKey[len(secretAPIKey)-3:])
}
