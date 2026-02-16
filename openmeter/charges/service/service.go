package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/charges"
)

type service struct {
	adapter charges.Adapter
}

type Config struct {
	Adapter charges.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	return nil
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: config.Adapter,
	}, nil
}

var _ charges.Service = (*service)(nil)
