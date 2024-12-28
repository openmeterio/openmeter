package appstripeentityapp

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/secret"
)

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase
	appstripeentity.AppData

	AppService             app.Service                         `json:"-"`
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

	if a.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if err := a.AppData.Validate(); err != nil {
		return fmt.Errorf("error validating stripe app data: %w", err)
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

	return nil
}
