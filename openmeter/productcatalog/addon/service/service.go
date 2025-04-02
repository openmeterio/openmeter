package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Config struct {
	Adapter   addon.Repository
	Feature   feature.FeatureConnector
	Logger    *slog.Logger
	Publisher eventbus.Publisher
}

func New(config Config) (addon.Service, error) {
	if config.Adapter == nil {
		return nil, errors.New("add-on adapter is required")
	}

	if config.Feature == nil {
		return nil, errors.New("feature connector is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.Publisher == nil {
		return nil, errors.New("publisher is required")
	}

	return &service{
		adapter:   config.Adapter,
		feature:   config.Feature,
		logger:    config.Logger,
		publisher: config.Publisher,
	}, nil
}

var _ addon.Service = (*service)(nil)

type service struct {
	adapter   addon.Repository
	feature   feature.FeatureConnector
	logger    *slog.Logger
	publisher eventbus.Publisher
}
