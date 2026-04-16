package appstripe

import (
	"context"
	"fmt"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// getStripeClient gets the Stripe client for the app
func (a App) getStripeClient(ctx context.Context, logOperation string, logFields ...any) (AppData, stripeclient.StripeAppClient, error) {
	// Get Stripe App
	stripeAppData, err := a.StripeAppService.GetStripeAppData(ctx, GetStripeAppDataInput{
		AppID: a.GetID(),
	})
	if err != nil {
		return AppData{}, nil, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.SecretService.GetAppSecret(ctx, secretentity.NewSecretID(a.GetID(), stripeAppData.APIKey.ID, APIKeySecretKey))
	if err != nil {
		return AppData{}, nil, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.StripeAppClientFactory(stripeclient.StripeAppClientConfig{
		AppID:      a.GetID(),
		AppService: a.AppService,
		APIKey:     apiKeySecret.Value,
		Logger:     a.Logger.With("operation", logOperation).With(logFields...),
	})
	if err != nil {
		return AppData{}, nil, fmt.Errorf("failed to create stripe client: %w", err)
	}

	return stripeAppData, stripeClient, nil
}
