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
)

// Register registers the stripe app with the marketplace
func Register(marketplace app.MarketplaceService, appService app.Service, db *entdb.Client) error {
	stripeAppFactory, err := NewAppFactory(AppFactoryConfig{
		AppService: appService,
		Client:     db,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe app factory: %w", err)
	}

	err = marketplace.Register(appentity.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: stripeAppFactory,
	})
	if err != nil {
		return fmt.Errorf("failed to register stripe app: %w", err)
	}

	return nil
}

// AppFactory is the factory for creating stripe app instances
type AppFactory struct {
	AppService     app.Service
	BillingService billing.Service
	Client         *entdb.Client
}

type AppFactoryConfig struct {
	AppService app.Service
	Client     *entdb.Client
}

func (a AppFactoryConfig) Validate() error {
	if a.AppService == nil {
		return errors.New("app service is required")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	return nil
}

func NewAppFactory(config AppFactoryConfig) (AppFactory, error) {
	if err := config.Validate(); err != nil {
		return AppFactory{}, fmt.Errorf("invalid app factory config: %w", err)
	}

	return AppFactory{
		AppService: config.AppService,
		Client:     config.Client,
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
