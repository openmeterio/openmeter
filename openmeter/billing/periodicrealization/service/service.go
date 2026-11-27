package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Config struct {
	Adapter periodicrealization.Adapter
	Locker  *lockr.Locker
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Locker == nil {
		errs = append(errs, errors.New("locker is required"))
	}

	return errors.Join(errs...)
}

type service struct {
	adapter periodicrealization.Adapter
	locker  *lockr.Locker
}

func New(config Config) (periodicrealization.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: config.Adapter,
		locker:  config.Locker,
	}, nil
}
