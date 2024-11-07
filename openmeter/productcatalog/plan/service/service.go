package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

type Config struct {
	Feature feature.FeatureConnector

	Adapter plan.Repository
	Logger  *slog.Logger
}

func New(config Config) (plan.Service, error) {
	if config.Feature == nil {
		return nil, errors.New("feature connector is required")
	}

	if config.Adapter == nil {
		return nil, errors.New("plan adapter is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	return &service{
		feature: config.Feature,
		adapter: config.Adapter,
		logger:  config.Logger,
	}, nil
}

// TODO(chrisgacsal): use transactional client for adapter operations

var _ plan.Service = (*service)(nil)

type service struct {
	feature feature.FeatureConnector

	adapter plan.Repository

	logger *slog.Logger
}
