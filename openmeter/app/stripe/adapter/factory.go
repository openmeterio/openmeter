package appstripeadapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// This file implements the app.AppFactory interface
var _ appentity.AppFactory = (*adapter)(nil)

// NewApp implement the app.AppFactory interface and returns a Stripe App by extending the AppBase
func (a adapter) NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
	stripeApp, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{AppID: appBase.GetID()})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	app, err := a.mapAppStripeFromDB(appBase, stripeApp)
	if err != nil {
		return nil, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}

// NewApp implement the app.AppFactory interface and installs a Stripe App type
func (a adapter) InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Check if the Stripe API key is a test key
	livemode := true

	if strings.HasPrefix(input.APIKey, "sk_test") || strings.HasPrefix(input.APIKey, "rk_test") {
		livemode = false
	}

	// Get stripe client
	stripeClient, err := a.stripeClientFactory(stripeclient.StripeClientConfig{
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
	apiKeySecretID, err := a.secretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
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
	webhookSecretID, err := a.secretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
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

	app, err := a.CreateStripeApp(ctx, createStripeAppInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return app, nil
}

// UninstallApp uninstalls an app by id
func (a adapter) UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error {
	// Get Stripe App
	app, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: input,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.secretService.GetAppSecret(ctx, app.APIKey)
	if err != nil {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Create Stripe Client
	stripeClient, err := a.stripeClientFactory(stripeclient.StripeClientConfig{
		Namespace: app.APIKey.Namespace,
		APIKey:    apiKeySecret.Value,
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
