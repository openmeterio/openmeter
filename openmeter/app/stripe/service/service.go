package appservice

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/secret"
)

var _ appstripe.Service = (*Service)(nil)

type Service struct {
	adapter       appstripe.Adapter
	appService    app.Service
	secretService secret.Service
}

type Config struct {
	Adapter       appstripe.Adapter
	AppService    app.Service
	SecretService secret.Service
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	if c.SecretService == nil {
		return errors.New("secret service cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	service := &Service{
		adapter:       config.Adapter,
		appService:    config.AppService,
		secretService: config.SecretService,
	}

	// Register stripe app in marketplace
	err := config.AppService.RegisterMarketplaceListing(appentity.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: service,
	})
	if err != nil {
		return service, fmt.Errorf("failed to register stripe app to marketplace: %w", err)
	}

	return service, nil
}
