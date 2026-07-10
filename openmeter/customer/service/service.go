package customerservice

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ customer.Service = (*Service)(nil)

type Service struct {
	adapter                  customer.Adapter
	requestValidatorRegistry customer.RequestValidatorRegistry
	publisher                eventbus.Publisher
	logger                   *slog.Logger

	hooks models.ServiceHookRegistry[customer.Customer]
}

func (s *Service) RegisterHooks(hooks ...models.ServiceHook[customer.Customer]) {
	s.hooks.RegisterHooks(hooks...)
}

type Config struct {
	Adapter   customer.Adapter
	Publisher eventbus.Publisher
	Logger    *slog.Logger
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:                  config.Adapter,
		requestValidatorRegistry: customer.NewRequestValidatorRegistry(),
		publisher:                config.Publisher,
		logger:                   config.Logger,
		hooks:                    models.ServiceHookRegistry[customer.Customer]{},
	}, nil
}
