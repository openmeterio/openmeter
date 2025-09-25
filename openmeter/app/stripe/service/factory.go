package appservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// This file implements the app.AppFactory interface
var _ app.AppFactory = (*Service)(nil)

// NewApp implement the app.AppFactory interface and returns a Stripe App by extending the AppBase
func (s *Service) NewApp(ctx context.Context, appBase app.AppBase) (app.App, error) {
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
func (s *Service) InstallAppWithAPIKey(ctx context.Context, input app.AppFactoryInstallAppWithAPIKeyInput) (app.App, error) {
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
		Logger:    s.logger.With("operation", "installAppWithAPIKey", "namespace", input.Namespace, "app_name", input.Name),
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
	appID := app.AppID{Namespace: input.Namespace, ID: ulid.Make().String()}

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

	// Get webhook URL
	webhookURL, err := s.webhookURLGenerator.GetWebhookURL(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook url: %w", err)
	}

	// Setup webhook
	var stripeWebhookEndpoint stripeclient.StripeWebhookEndpoint
	if !s.disableWebhookRegistration {
		stripeWebhookEndpoint, err = stripeClient.SetupWebhook(ctx, stripeclient.SetupWebhookInput{
			AppID:      appID,
			WebhookURL: webhookURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to setup webhook: %w", err)
		}
	} else {
		// Let's generate a fake secret for development purposes
		stripeWebhookEndpoint = stripeclient.StripeWebhookEndpoint{
			EndpointID: "endpoint-registration-disabled",
			Secret:     "fake-secret",
		}
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
		CreateAppInput: app.CreateAppInput{
			ID:          &appID,
			Namespace:   input.Namespace,
			Name:        input.Name,
			Description: fmt.Sprintf("Stripe account %s", stripeAccount.StripeAccountID),
			Type:        app.AppTypeStripe,
		},

		StripeAccountID: stripeAccount.StripeAccountID,
		Livemode:        livemode,
		APIKey:          apiKeySecretID,
		MaskedAPIKey:    s.generateMaskedSecretAPIKey(input.APIKey),
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
func (s *Service) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	// Get Stripe App
	stripeApp, err := s.adapter.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: input,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Delete stripe customer data
	err = s.adapter.DeleteStripeCustomerData(ctx, appstripeentity.DeleteStripeCustomerDataInput{
		AppID: &input,
	})
	if err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	// Delete stripe app data
	err = s.adapter.DeleteStripeAppData(ctx, appstripeentity.DeleteStripeAppDataInput{
		AppID: input,
	})
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := s.secretService.GetAppSecret(ctx, stripeApp.APIKey)

	// If the secret is not found, we continue with the uninstallation
	var secretNotFoundError *secretentity.SecretNotFoundError

	if err != nil && !errors.As(err, &secretNotFoundError) {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Try to delete the webhook, it may fail if the token is invalid
	if err == nil {
		// Create Stripe Client
		stripeClient, err := s.adapter.GetStripeAppClientFactory()(stripeclient.StripeAppClientConfig{
			AppID:      input,
			AppService: s.appService,
			APIKey:     apiKeySecret.Value,
			Logger:     s.logger.With("operation", "uninstalApp", "app_id", input.ID),
		})
		if err != nil {
			return fmt.Errorf("failed to create stripe client")
		}

		// Delete Webhook
		err = stripeClient.DeleteWebhook(ctx, stripeclient.DeleteWebhookInput{
			AppID:           input,
			StripeWebhookID: stripeApp.StripeWebhookID,
		})

		// If the error is not an authentication error, we return it
		if app.IsAppProviderAuthenticationError(err) {
			return fmt.Errorf("failed to delete stripe webhook")
		}
	}

	// Delete secrets
	if err := s.secretService.DeleteAppSecret(ctx, stripeApp.APIKey); err != nil && !errors.As(err, &secretNotFoundError) {
		return fmt.Errorf("failed to delete stripe api key secret")
	}

	if err := s.secretService.DeleteAppSecret(ctx, stripeApp.WebhookSecret); err != nil && !errors.As(err, &secretNotFoundError) {
		return fmt.Errorf("failed to delete stripe webhook secret")
	}

	return nil
}

// newApp combines the app base and stripe app data to create a new app
func (s *Service) newApp(appBase app.AppBase, stripeApp appstripeentity.AppData) (appstripeentityapp.App, error) {
	app := appstripeentityapp.App{
		Meta: appstripeentityapp.Meta{
			AppBase: appBase,
			AppData: stripeApp,
		},
		AppService:             s.appService,
		BillingService:         s.billingService,
		StripeAppService:       s,
		SecretService:          s.secretService,
		StripeAppClientFactory: s.adapter.GetStripeAppClientFactory(),
		Logger:                 s.logger,
	}

	if err := app.Validate(); err != nil {
		return appstripeentityapp.App{}, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}
