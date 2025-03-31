package customerservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var _ customer.Service = (*Service)(nil)

type Service struct {
	adapter                  customer.Adapter
	entitlementConnector     entitlement.Connector
	requestValidatorRegistry customer.RequestValidatorRegistry
	publisher                eventbus.Publisher
}

type Config struct {
	Adapter              customer.Adapter
	EntitlementConnector entitlement.Connector
	Publisher            eventbus.Publisher
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.EntitlementConnector == nil {
		return errors.New("entitlement connector cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:                  config.Adapter,
		entitlementConnector:     config.EntitlementConnector,
		requestValidatorRegistry: customer.NewRequestValidatorRegistry(),
		publisher:                config.Publisher,
	}, nil
}
