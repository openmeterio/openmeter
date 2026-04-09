package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Config struct {
	Adapter     creditpurchase.Adapter
	Handler     creditpurchase.Handler
	Lineage     lineage.Service
	MetaAdapter meta.Adapter
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("credit purchase handler cannot be null"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service cannot be null"))
	}

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (creditpurchase.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:     config.Adapter,
		handler:     config.Handler,
		lineage:     config.Lineage,
		metaAdapter: config.MetaAdapter,
	}, nil
}

type service struct {
	adapter     creditpurchase.Adapter
	metaAdapter meta.Adapter
	handler     creditpurchase.Handler
	lineage     lineage.Service
}
