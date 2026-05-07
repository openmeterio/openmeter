package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

var _ taxcode.Service = (*Service)(nil)

type Service struct {
	adapter taxcode.TaxCodeRepository
	logger  *slog.Logger
}

type Config struct {
	Adapter taxcode.TaxCodeRepository
	Logger  *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		logger:  config.Logger,
	}, nil
}
