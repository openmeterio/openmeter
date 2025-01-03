package appservice

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// This file implements the app.AppFactory interface
var _ appentity.AppFactory = (*Service)(nil)

// NewApp implement the app.AppFactory interface and returns a Stripe App by extending the AppBase
func (s *Service) NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
	stripeApp, err := s.adapter.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{AppID: appBase.GetID()})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	app, err := s.newApp(appBase, stripeApp)
	if err != nil {
		return nil, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}

// NewApp implement the app.AppFactory interface and installs a Stripe App type
func (s *Service) InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Check if the Stripe API key is a test key
	livemode := stripeclient.IsAPIKeyLiveMode(input.APIKey)

	// Get stripe client
	stripeClient, err := s.adapter.GetStripeClientFactory()(stripeclient.StripeClientConfig{
		Namespace: input.Namespace,
		APIKey:    input.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Retrieve stripe account
	stripeAccount, err := stripeClient.GetAccount(ctx)
	if err != nil {
		return nil, err
	}

	// We generate the app ID here because we need it to setup the webhook and create the secrets
	appID := appentitybase.AppID{Namespace: input.Namespace, ID: ulid.Make().String()}

	// TODO: secret creation, webhook setup and app creation should be done in a transaction
	// This is challenging because we need to coordinate between three remote services (secret, stripe, db)

	// Create API Key secret
	apiKeySecretID, err := s.secretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
		AppID: appID,
		Key:   appstripeentity.APIKeySecretKey,
		Value: input.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	// Setup webhook
	stripeWebhookEndpoint, err := stripeClient.SetupWebhook(ctx, stripeclient.SetupWebhookInput{
		AppID:   appID,
		BaseURL: input.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup webhook: %w", err)
	}

	// Create webhook secret
	webhookSecretID, err := s.secretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
		AppID: appID,
		Key:   appstripeentity.WebhookSecretKey,
		Value: stripeWebhookEndpoint.Secret,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	// Create stripe app
	createStripeAppInput := appstripeentity.CreateAppStripeInput{
		ID:              &appID.ID,
		Namespace:       input.Namespace,
		Name:            input.Name,
		Description:     fmt.Sprintf("Stripe account %s", stripeAccount.StripeAccountID),
		StripeAccountID: stripeAccount.StripeAccountID,
		Livemode:        livemode,
		APIKey:          apiKeySecretID,
		StripeWebhookID: stripeWebhookEndpoint.EndpointID,
		WebhookSecret:   webhookSecretID,
	}

	if err := createStripeAppInput.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create stripe app input: %w", err)
	}

	stripeApp, err := s.adapter.CreateStripeApp(ctx, createStripeAppInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	app, err := s.newApp(stripeApp.AppBase, stripeApp.AppData)
	if err != nil {
		return nil, fmt.Errorf("failed to factor stripe app: %w", err)
	}

	return app, nil
}

// UninstallApp uninstalls an app by id
func (s *Service) UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error {
	// Get Stripe App
	app, err := s.adapter.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: input,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := s.secretService.GetAppSecret(ctx, app.APIKey)
	if err != nil {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Create Stripe Client
	stripeClient, err := s.adapter.GetStripeAppClientFactory()(stripeclient.StripeAppClientConfig{
		AppID:      input,
		AppService: s.appService,
		APIKey:     apiKeySecret.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe client")
	}

	// Delete Webhook
	err = stripeClient.DeleteWebhook(ctx, stripeclient.DeleteWebhookInput{
		AppID:           input,
		StripeWebhookID: app.StripeWebhookID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete stripe webhook")
	}

	// We don't need to delete Stripe specific rows from DB because of cascade delete in app.

	return nil
}

// newApp combines the app base and stripe app data to create a new app
func (s *Service) newApp(appBase appentitybase.AppBase, stripeApp appstripeentity.AppData) (appstripeentityapp.App, error) {
	app := appstripeentityapp.App{
		AppBase:                appBase,
		AppData:                stripeApp,
		AppService:             s.appService,
		StripeAppService:       s,
		SecretService:          s.secretService,
		StripeAppClientFactory: s.adapter.GetStripeAppClientFactory(),
	}

	if err := app.Validate(); err != nil {
		return appstripeentityapp.App{}, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}
