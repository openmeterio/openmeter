package billingservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var _ billing.Service = (*Service)(nil)

type Service struct {
	adapter         billing.Adapter
	customerService customer.CustomerService
	appService      app.Service
}

type Config struct {
	Adapter         billing.Adapter
	CustomerService customer.CustomerService
	AppService      app.Service
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.CustomerService == nil {
		return errors.New("customer service cannot be null")
	}

	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:         config.Adapter,
		customerService: config.CustomerService,
		appService:      config.AppService,
	}, nil
}
