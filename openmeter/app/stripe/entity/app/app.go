package appstripeentityapp

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/secret"
)

type Meta struct {
	app.AppBase
	appstripeentity.AppData
}

var _ app.EventAppParser = (*Meta)(nil)

func (m *Meta) FromEventAppData(event app.EventApp) error {
	m.AppBase = event.AppBase

	if err := event.AppData.ParseInto(&m.AppData); err != nil {
		return fmt.Errorf("error parsing app data: %w", err)
	}

	return nil
}

// App represents an installed Stripe app
type App struct {
	Meta

	Logger *slog.Logger `json:"-"`

	AppService             app.Service                         `json:"-"`
	BillingService         billing.Service                     `json:"-"`
	StripeAppClientFactory stripeclient.StripeAppClientFactory `json:"-"`
	StripeAppService       stripeapp.Service                   `json:"-"`
	SecretService          secret.Service                      `json:"-"`
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if err := a.AppData.Validate(); err != nil {
		return fmt.Errorf("error validating stripe app data: %w", err)
	}

	if a.Type != app.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if err := a.AppData.Validate(); err != nil {
		return fmt.Errorf("error validating stripe app data: %w", err)
	}

	if a.BillingService == nil {
		return errors.New("billing service is required")
	}

	if a.StripeAppClientFactory == nil {
		return errors.New("stripe client factory is required")
	}

	if a.AppService == nil {
		return errors.New("app service is required")
	}

	if a.StripeAppService == nil {
		return errors.New("stripe app service is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	if a.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func (a App) GetEventAppData() (app.EventAppData, error) {
	return app.NewEventAppData(a.AppData)
}
