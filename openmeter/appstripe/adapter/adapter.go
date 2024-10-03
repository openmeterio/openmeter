package appstripeadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client              *entdb.Client
	AppService          app.Service
	CustomerService     customer.Service
	Marketplace         app.MarketplaceService
	SecretService       secret.Service
	StripeClientFactory func(apiKey string) StripeClient
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.AppService == nil {
		return errors.New("app service is required")
	}

	if c.CustomerService == nil {
		return errors.New("customer service is required")
	}

	if c.Marketplace == nil {
		return errors.New("marketplace is required")
	}

	if c.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

func New(config Config) (appstripe.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	adapter := &adapter{
		db:              config.Client,
		appService:      config.AppService,
		customerService: config.CustomerService,
	}

	stripeClientFactory := config.StripeClientFactory
	if stripeClientFactory == nil {
		stripeClientFactory = StripeClientFactory
	}

	stripeAppFactory, err := NewAppFactory(AppFactoryConfig{
		AppService:          config.AppService,
		AppStripeAdapter:    adapter,
		Client:              config.Client,
		SecretService:       config.SecretService,
		StripeClientFactory: stripeClientFactory,
	})
	if err != nil {
		return adapter, fmt.Errorf("failed to create stripe app factory: %w", err)
	}

	err = config.Marketplace.Register(appentity.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: stripeAppFactory,
	})
	if err != nil {
		return adapter, fmt.Errorf("failed to register stripe app: %w", err)
	}

	return adapter, nil
}

var _ appstripe.Adapter = (*adapter)(nil)

type adapter struct {
	db *entdb.Client

	appService      app.Service
	customerService customer.Service
}

// Tx implements entutils.TxCreator interface
func (a adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}
