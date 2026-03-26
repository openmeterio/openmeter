package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Adapter     flatfee.Adapter
	Handler     flatfee.Handler
	MetaAdapter meta.Adapter
	Locker      *lockr.Locker
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

	if c.Locker == nil {
		errs = append(errs, errors.New("locker cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (flatfee.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:     config.Adapter,
		handler:     config.Handler,
		metaAdapter: config.MetaAdapter,
		locker:      config.Locker,
	}, nil
}

type service struct {
	adapter     flatfee.Adapter
	handler     flatfee.Handler
	metaAdapter meta.Adapter
	locker      *lockr.Locker
}
