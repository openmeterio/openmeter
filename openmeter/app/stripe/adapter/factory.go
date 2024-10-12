package appstripeadapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// This file implements the app.AppFactory interface
var _ appentity.AppFactory = (*adapter)(nil)

// NewApp implement the app.AppFactory interface and returns a Stripe App by extending the AppBase
func (a adapter) NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
	stripeApp, err := a.db.AppStripe.
		Query().
		Where(appstripedb.ID(appBase.GetID().ID)).
		Where(appstripedb.Namespace(appBase.GetID().Namespace)).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, app.AppNotFoundError{
				AppID: appBase.GetID(),
			}
		}

		return nil, fmt.Errorf("failed to get stripe app: %w", err)
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
		Name:            "Stripe",
		Description:     fmt.Sprintf("Stripe account %s", stripeAccount.StripeAccountID),
		StripeAccountID: stripeAccount.StripeAccountID,
		Livemode:        livemode,
		APIKey:          apiKeySecretID,
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
