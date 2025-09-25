package appservice

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var _ appstripe.Service = (*Service)(nil)

type Service struct {
	adapter                    appstripe.Adapter
	appService                 app.Service
	secretService              secret.Service
	billingService             billing.Service
	logger                     *slog.Logger
	publisher                  eventbus.Publisher
	disableWebhookRegistration bool
	webhookURLGenerator        app.WebhookURLGenerator
}

type Config struct {
	Adapter                    appstripe.Adapter
	AppService                 app.Service
	SecretService              secret.Service
	BillingService             billing.Service
	Logger                     *slog.Logger
	DisableWebhookRegistration bool
	Publisher                  eventbus.Publisher
	WebhookURLGenerator        app.WebhookURLGenerator
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

	if c.BillingService == nil {
		return errors.New("billing service cannot be null")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	if c.WebhookURLGenerator == nil {
		return errors.New("webhook url generator cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	service := &Service{
		adapter:                    config.Adapter,
		appService:                 config.AppService,
		secretService:              config.SecretService,
		billingService:             config.BillingService,
		logger:                     config.Logger,
		disableWebhookRegistration: config.DisableWebhookRegistration,
		publisher:                  config.Publisher,
		webhookURLGenerator:        config.WebhookURLGenerator,
	}

	// Register stripe app in marketplace
	err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{
		Listing: appstripeentity.StripeMarketplaceListing,
		Factory: service,
	})
	if err != nil {
		return service, fmt.Errorf("failed to register stripe app to marketplace: %w", err)
	}

	return service, nil
}
