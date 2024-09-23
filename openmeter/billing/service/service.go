package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.Service = (*Service)(nil)

type Service struct {
	repo billing.Repository
}

type Config struct {
	Repository billing.Repository
}

func (c Config) Validate() error {
	if c.Repository == nil {
		return errors.New("repository cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		repo: config.Repository,
	}, nil
}
