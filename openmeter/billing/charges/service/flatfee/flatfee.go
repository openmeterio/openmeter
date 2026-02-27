package flatfee

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type Config struct {
	Adapter        charges.Adapter
	FlatFeeHandler charges.FlatFeeHandler
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.FlatFeeHandler == nil {
		errs = append(errs, errors.New("flat fee handler cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (charges.FlatFeeService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:        config.Adapter,
		flatFeeHandler: config.FlatFeeHandler,
	}, nil
}

type service struct {
	adapter        charges.Adapter
	flatFeeHandler charges.FlatFeeHandler
}
