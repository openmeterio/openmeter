package service

import (
	"context"

	"github.com/invopop/gobl/currency"

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

func (s *Service) ListCurrencies(ctx context.Context) ([]currencies.Currency, error) {
	return s.adapter.ListCurrencies(ctx)
}

func (s *Service) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (*currency.Def, error) {
	return s.adapter.CreateCurrency(ctx, params)
}

func (s *Service) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (*currencies.CostBasis, error) {
	return s.adapter.CreateCostBasis(ctx, params)
}

func (s *Service) GetCostBasesByCurrencyID(ctx context.Context, currencyID string) ([]currencies.CostBasis, error) {
	return s.adapter.GetCostBasesByCurrencyID(ctx, currencyID)
}
