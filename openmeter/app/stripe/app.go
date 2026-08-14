package appstripe

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/openmeter/secret"
)

type Meta struct {
	app.AppBase
	AppData
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
	Operations
}

type Operations interface {
	app.AppOperations
	customerapp.App
	billing.InvoicingApp
}

type appOperations struct {
	Meta

	Logger *slog.Logger `json:"-"`

	AppService             app.Service                         `json:"-"`
	BillingService         billing.Service                     `json:"-"`
	StripeAppClientFactory stripeclient.StripeAppClientFactory `json:"-"`
	StripeAppService       Service                             `json:"-"`
	SecretService          secret.Service                      `json:"-"`
}

var _ Operations = (*appOperations)(nil)

func (a appOperations) Validate() error {
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

func (a appOperations) ValidateCapabilities(capabilities ...app.CapabilityType) error {
	return a.AppBase.ValidateCapabilities(capabilities...)
}

func (a App) GetEventAppData() (app.EventAppData, error) {
	return app.NewEventAppData(a.AppData)
}

type AppConfig struct {
	Logger                 *slog.Logger
	AppService             app.Service
	BillingService         billing.Service
	StripeAppClientFactory stripeclient.StripeAppClientFactory
	StripeAppService       Service
	SecretService          secret.Service
}

func New(meta Meta, config AppConfig) (App, error) {
	implementation := &appOperations{
		Meta:                   meta,
		Logger:                 config.Logger,
		AppService:             config.AppService,
		BillingService:         config.BillingService,
		StripeAppClientFactory: config.StripeAppClientFactory,
		StripeAppService:       config.StripeAppService,
		SecretService:          config.SecretService,
	}

	if err := implementation.Validate(); err != nil {
		return App{}, err
	}

	return App{
		Meta:       meta,
		Operations: implementation,
	}, nil
}

func NewDeleted(meta Meta) App {
	return App{
		Meta: meta,
		Operations: &deletedApp{
			DeletedApp:          app.NewDeletedApp(meta.GetID()),
			DeletedInvoicingApp: billing.NewDeletedInvoicingApp(meta.GetID()),
		},
	}
}

type deletedApp struct {
	app.DeletedApp
	billing.DeletedInvoicingApp
}

var _ Operations = (*deletedApp)(nil)
