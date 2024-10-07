package appservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.GetWebhookSecretOutput{}, appstripe.ValidationError{
			Err: fmt.Errorf("error get webhook secret: %w", err),
		}
	}

	return s.adapter.GetWebhookSecret(ctx, input)
}

func (s *Service) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCheckoutSession{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create checkout session: %w", err),
		}
	}

	return s.adapter.CreateCheckoutSession(ctx, input)
}

func (s *Service) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, appstripe.ValidationError{
			Err: fmt.Errorf("error set customer default payment method: %w", err),
		}
	}

	return s.adapter.SetCustomerDefaultPaymentMethod(ctx, input)
}
