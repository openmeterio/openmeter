package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ splitlinegroup.Service = (*Service)(nil)
	_ models.Validator       = (*Config)(nil)
)

type Config struct {
	Adapter splitlinegroup.Adapter
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Service struct {
	adapter splitlinegroup.Adapter
	logger  *slog.Logger
}

func New(config Config) (splitlinegroup.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		logger:  config.Logger,
	}, nil
}
