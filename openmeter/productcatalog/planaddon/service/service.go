package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Config struct {
	Adapter   planaddon.Repository
	Plan      plan.Service
	Addon     addon.Service
	Logger    *slog.Logger
	Publisher eventbus.Publisher
}

func New(config Config) (planaddon.Service, error) {
	if config.Adapter == nil {
		return nil, errors.New("add-on assignment adapter is required")
	}

	if config.Plan == nil {
		return nil, errors.New("plan service is required")
	}

	if config.Addon == nil {
		return nil, errors.New("add-on service is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.Publisher == nil {
		return nil, errors.New("publisher is required")
	}

	return &service{
		adapter:   config.Adapter,
		plan:      config.Plan,
		addon:     config.Addon,
		logger:    config.Logger,
		publisher: config.Publisher,
	}, nil
}

var _ planaddon.Service = (*service)(nil)

type service struct {
	adapter   planaddon.Repository
	plan      plan.Service
	addon     addon.Service
	logger    *slog.Logger
	publisher eventbus.Publisher
}
