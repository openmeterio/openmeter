package appstripe

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
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

	marketplace.Register(appentity.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: stripeAppFactory,
	})

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

func (f AppFactory) NewApp(ctx context.Context, app appentity.AppBase) (appentity.App, error) {
	return &appstripeentity.App{
		AppBase: app,
		Client:  f.Client,
	}, nil
}
