package service

import (
	"context"

	v3 "github.com/openmeterio/openmeter/api/v3"
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

func (s *Service) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]v3.BillingCurrency, int, error) {
	return s.adapter.ListCurrencies(ctx, params)
}

func (s *Service) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (v3.BillingCurrencyCustom, error) {
	return s.adapter.CreateCurrency(ctx, params)
}

func (s *Service) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (v3.BillingCostBasis, error) {
	return s.adapter.CreateCostBasis(ctx, params)
}

func (s *Service) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) ([]v3.BillingCostBasis, int, error) {
	return s.adapter.ListCostBases(ctx, params)
}
