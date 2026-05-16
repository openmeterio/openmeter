package service

import (
	"errors"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type Config struct {
	Adapter   plan.Repository
	TaxCode   taxcode.Service
	Logger    *slog.Logger
	Publisher eventbus.Publisher

	FeatureResolver productcatalog.FeatureResolver
}

func New(config Config) (plan.Service, error) {
	if config.Adapter == nil {
		return nil, errors.New("plan adapter is required")
	}

	if config.FeatureResolver == nil {
		return nil, errors.New("feature resolver is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.TaxCode == nil {
		return nil, errors.New("tax code service is required")
	}

	if config.Publisher == nil {
		return nil, errors.New("publisher is required")
	}

	return &service{
		adapter:   config.Adapter,
		taxCode:   config.TaxCode,
		logger:    config.Logger,
		publisher: config.Publisher,

		featureResolver: config.FeatureResolver,
	}, nil
}

var _ plan.Service = (*service)(nil)

type service struct {
	adapter   plan.Repository
	taxCode   taxcode.Service
	logger    *slog.Logger
	publisher eventbus.Publisher

	featureResolver productcatalog.FeatureResolver
}
