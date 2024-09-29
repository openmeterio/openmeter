package appcustomerservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/appcustomer"
)

var _ appcustomer.Service = (*Service)(nil)

type Service struct {
	adapter appcustomer.Adapter
}

type Config struct {
	Adapter appcustomer.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
	}, nil
}
