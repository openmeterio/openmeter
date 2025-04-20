package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/ledger"
)

var _ ledger.Service = (*Service)(nil)

type Config struct {
	Adapter ledger.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	return nil
}

type Service struct {
	adapter ledger.Adapter
}

func NewService(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{adapter: config.Adapter}, nil
}
