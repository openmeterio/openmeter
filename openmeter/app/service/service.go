package appservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ app.Service = (*Service)(nil)

type Service struct {
	adapter   app.Adapter
	publisher eventbus.Publisher

	hooks models.ServiceHookRegistry[app.AppBase]
}

func (s *Service) RegisterHooks(hooks ...models.ServiceHook[app.AppBase]) {
	s.hooks.RegisterHooks(hooks...)
}

type Config struct {
	Adapter   app.Adapter
	Publisher eventbus.Publisher
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:   config.Adapter,
		publisher: config.Publisher,
		hooks:     models.ServiceHookRegistry[app.AppBase]{},
	}, nil
}
