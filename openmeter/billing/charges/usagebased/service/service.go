package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type Config struct {
	Adapter     usagebased.Adapter
	Handler     usagebased.Handler
	MetaAdapter meta.Adapter
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler cannot be null"))
	}

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (usagebased.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:     config.Adapter,
		handler:     config.Handler,
		metaAdapter: config.MetaAdapter,
	}, nil
}

type service struct {
	adapter     usagebased.Adapter
	handler     usagebased.Handler
	metaAdapter meta.Adapter
}
