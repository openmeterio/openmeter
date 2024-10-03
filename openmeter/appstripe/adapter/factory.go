package appstripeadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	StripeClientFactory func(apiKey string) StripeClient
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
	StripeClientFactory func(apiKey string) StripeClient
}

func NewAppFactory(config AppFactoryConfig) (AppFactory, error) {
	if err := config.Validate(); err != nil {
		return AppFactory{}, fmt.Errorf("invalid app factory config: %w", err)
	}

	return AppFactory{
		AppService:          config.AppService,
		AppStripeAdapter:    config.AppStripeAdapter,
		SecretService:       config.SecretService,
		Client:              config.Client,
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

	return &appstripeentity.App{
		AppBase: appBase,
		Client:  f.Client,

		StripeAccountId: stripeApp.StripeAccountID,
		Livemode:        stripeApp.StripeLivemode,
	}, nil
}

func (f AppFactory) InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Check if the API key is a test key
	livemode := true

	if strings.HasPrefix(input.APIKey, "sk_test") || strings.HasPrefix(input.APIKey, "rk_test") {
		livemode = false
	}

	stripeClient := f.StripeClientFactory(input.APIKey)

	// TODO: this is the first call to stripe, we should check if the API key is valid and return typed errors
	// Get stripe account
	stripeAccount, err := stripeClient.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe account: %w", err)
	}

	// Create secret
	secretID, err := f.SecretService.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{
		Namespace: input.Namespace,
		Key:       appstripeentity.APIKeySecretKey,
		Value:     input.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	// Create stripe app
	createStripeAppInput := appstripeentity.CreateAppStripeInput{
		Namespace:       input.Namespace,
		Name:            "Stripe",
		Description:     "Stripe",
		StripeAccountID: stripeAccount.StripeAccountID,
		Livemode:        livemode,
		APIKey:          secretID,
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
