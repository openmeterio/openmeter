package appstripeadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type AppFactoryConfig struct {
	AppService          app.Service
	AppStripeAdapter    appstripe.Adapter
	Client              *entdb.Client
	SecretService       secret.Service
	StripeClientFactory appstripeentity.StripeClientFactory
}

func (a AppFactoryConfig) Validate() error {
	if a.AppService == nil {
		return errors.New("app service is required")
	}

	if a.AppStripeAdapter == nil {
		return errors.New("app stripe adapter is required")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	if a.StripeClientFactory == nil {
		return errors.New("stripe client factory is required")
	}

	return nil
}

// AppFactory is the factory for creating stripe app instances
type AppFactory struct {
	AppService          app.Service
	AppStripeAdapter    appstripe.AppStripeAdapter
	BillingService      billing.Service
	Client              *entdb.Client
	SecretService       secret.Service
	StripeClientFactory appstripeentity.StripeClientFactory
}

func NewAppFactory(config AppFactoryConfig) (AppFactory, error) {
	if err := config.Validate(); err != nil {
		return AppFactory{}, fmt.Errorf("invalid app factory config: %w", err)
	}

	return AppFactory{
		AppService:          config.AppService,
		AppStripeAdapter:    config.AppStripeAdapter,
		Client:              config.Client,
		SecretService:       config.SecretService,
		StripeClientFactory: config.StripeClientFactory,
	}, nil
}

func (f AppFactory) NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
	stripeApp, err := f.Client.AppStripe.
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

	app, err := mapAppStripeFromDB(appBase, stripeApp, f.Client, f.SecretService, f.StripeClientFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}

func (f AppFactory) InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error) {
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
	stripeClient, err := f.StripeClientFactory(appstripeentity.StripeClientConfig{
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
	apiKeySecretID, err := f.SecretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
		AppID: appID,
		Key:   appstripeentity.APIKeySecretKey,
		Value: input.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	// Setup webhook
	stripeWebhookEndpoint, err := stripeClient.SetupWebhook(ctx, appstripeentity.StripeClientSetupWebhookInput{
		AppID: appID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup webhook: %w", err)
	}

	// Create webhook secret
	webhookSecretID, err := f.SecretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
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

	app, err := f.AppStripeAdapter.CreateStripeApp(ctx, createStripeAppInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return app, nil
}
