package service

import (
	"context"

	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/samber/lo"
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
	customCurrencies, err := s.adapter.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	defs := lo.Map(lo.Filter(
		currency.Definitions(),
		func(def *currency.Def, _ int) bool {
			// NOTE: this filters out non-iso currencies such as crypto
			return def.ISONumeric != ""
		},
	), func(def *currency.Def, _ int) currencies.Currency {
		return currencies.Currency{
			Code:                 def.ISOCode.String(),
			Name:                 def.Name,
			Symbol:               def.Symbol,
			SmallestDenomination: int8(def.SmallestDenomination),
			DisambiguateSymbol:   def.DisambiguateSymbol,
			Subunits:             uint32(def.Subunits),
			IsCustom:             false,
		}
	})

	return lo.Map(append(customCurrencies, defs...), func(def currencies.Currency, _ int) currencies.Currency {
		return currencies.Currency{
			ID:                   def.ID,
			Code:                 def.Code,
			Name:                 def.Name,
			Symbol:               def.Symbol,
			SmallestDenomination: int8(def.SmallestDenomination),
			DisambiguateSymbol:   def.DisambiguateSymbol,
			Subunits:             uint32(def.Subunits),
			IsCustom:             def.IsCustom,
		}
	}), nil
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
