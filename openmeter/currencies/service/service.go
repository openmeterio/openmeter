package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/currencies"
)

var _ currencies.CurrencyService = (*Service)(nil)

type Service struct {
	adapter currencies.Adapter
}

func New(adapter currencies.Adapter) *Service {
	return &Service{
		adapter: adapter,
	}
}

func (s *Service) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]currencies.Currency, int, error) {
	return s.adapter.ListCurrencies(ctx, params)
}

func (s *Service) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return s.adapter.CreateCurrency(ctx, params)
}

func (s *Service) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (*currencies.CostBasis, error) {
	return s.adapter.CreateCostBasis(ctx, params)
}

func (s *Service) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) ([]currencies.CostBasis, int, error) {
	return s.adapter.ListCostBases(ctx, params)
}
