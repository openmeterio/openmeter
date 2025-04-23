package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ appcustominvoicing.Service = (*Service)(nil)

type Service struct {
	adapter appcustominvoicing.Adapter
	logger  *slog.Logger

	// dependencies
	appService     app.Service
	billingService billing.Service
}

type Config struct {
	Adapter appcustominvoicing.Adapter
	Logger  *slog.Logger

	AppService     app.Service
	BillingService billing.Service
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be nil")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be nil")
	}

	if c.AppService == nil {
		return errors.New("app service cannot be nil")
	}

	if c.BillingService == nil {
		return errors.New("billing service cannot be nil")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:        config.Adapter,
		logger:         config.Logger,
		appService:     config.AppService,
		billingService: config.BillingService,
	}, nil
}
