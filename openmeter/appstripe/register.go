package appstripe

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type RegisterConfig struct {
	AppService       app.Service
	AppStripeService Service
	Client           *entdb.Client
	Marketplace      app.MarketplaceService
	SecretService    secret.Service
}

func (c RegisterConfig) Validate() error {
	if c.AppService == nil {
		return errors.New("app service is required")
	}

	if c.AppStripeService == nil {
		return errors.New("app stripe service is required")
	}

	if c.Client == nil {
		return errors.New("client is required")
	}

	if c.Marketplace == nil {
		return errors.New("marketplace is required")
	}

	if c.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

// Register registers the stripe app with the marketplace
func Register(config RegisterConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid register config: %w", err)
	}

	stripeAppFactory, err := NewAppFactory(AppFactoryConfig{
		AppService:       config.AppService,
		AppStripeService: config.AppStripeService,
		Client:           config.Client,
		SecretService:    config.SecretService,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe app factory: %w", err)
	}

	err = config.Marketplace.Register(appentity.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: stripeAppFactory,
	})
	if err != nil {
		return fmt.Errorf("failed to register stripe app: %w", err)
	}

	return nil
}

type AppFactoryConfig struct {
	AppService       app.Service
	AppStripeService Service
	SecretService    secret.Service
	Client           *entdb.Client
}

func (a AppFactoryConfig) Validate() error {
	if a.AppService == nil {
		return errors.New("app service is required")
	}

	if a.AppStripeService == nil {
		return errors.New("app stripe service is required")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

// AppFactory is the factory for creating stripe app instances
type AppFactory struct {
	AppService       app.Service
	AppStripeService Service
	BillingService   billing.Service
	SecretService    secret.Service
	Client           *entdb.Client
}

func NewAppFactory(config AppFactoryConfig) (AppFactory, error) {
	if err := config.Validate(); err != nil {
		return AppFactory{}, fmt.Errorf("invalid app factory config: %w", err)
	}

	return AppFactory{
		AppService:       config.AppService,
		AppStripeService: config.AppStripeService,
		SecretService:    config.SecretService,
		Client:           config.Client,
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

	// TODO: get accountID from Stripe
	stripeAccountID := "todo_stripe_account_id"
	livemode := false

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
	app, err := f.AppStripeService.CreateStripeApp(ctx, appstripeentity.CreateAppStripeInput{
		Namespace:       input.Namespace,
		Name:            "Stripe",
		Description:     "Stripe",
		StripeAccountID: stripeAccountID,
		Livemode:        livemode,
		APIKey:          secretID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return app, nil
}
