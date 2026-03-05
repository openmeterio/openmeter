package costservice

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/cost"
)

var _ cost.Service = (*Service)(nil)

type Service struct {
	adapter cost.Adapter
}

type Config struct {
	Adapter cost.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be nil")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
	}, nil
}

func (s *Service) QueryFeatureCost(ctx context.Context, input cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return s.adapter.QueryFeatureCost(ctx, input)
}
