package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ taxcode.Service = (*Service)(nil)

type Service struct {
	adapter taxcode.Repository
	logger  *slog.Logger

	hooks models.ServiceHookRegistry[taxcode.TaxCode]
}

type Config struct {
	Adapter taxcode.Repository
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

func (s *Service) RegisterHooks(hooks ...models.ServiceHook[taxcode.TaxCode]) {
	s.hooks.RegisterHooks(hooks...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		logger:  config.Logger,
		hooks:   models.ServiceHookRegistry[taxcode.TaxCode]{},
		// TODO: add event publisher
	}, nil
}
