package service

import (
	"context"
	"fmt"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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

func (s *Service) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	if params.Validate() != nil {
		return pagination.Result[currencies.Currency]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate()))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[currencies.Currency], error) {
		includeCustom := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeCustom
		includeFiat := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeFiat

		// Custom-only: delegate pagination entirely to the adapter (DB-level)
		if includeCustom && !includeFiat {
			return s.adapter.ListCustomCurrencies(ctx, params)
		}

		// Fiat-only or combined: enumerate in-memory then paginate
		var items []currencies.Currency

		if includeCustom {
			// Fetch all custom currencies without DB pagination for in-memory merge
			allParams := params
			allParams.Page = pagination.Page{}
			customResult, err := s.adapter.ListCustomCurrencies(ctx, allParams)
			if err != nil {
				return pagination.Result[currencies.Currency]{}, err
			}
			items = append(items, customResult.Items...)
		}

		if includeFiat {
			for _, def := range lo.Filter(currency.Definitions(), func(def *currency.Def, _ int) bool {
				// NOTE: this filters out non-iso currencies such as crypto
				return def.ISONumeric != ""
			}) {
				items = append(items, currencies.Currency{
					Code:   def.ISOCode.String(),
					Name:   def.Name,
					Symbol: def.Symbol,
				})
			}
		}

		total := len(items)

		pageSize := params.Page.PageSize
		pageNumber := params.Page.PageNumber
		if pageSize > 0 && pageNumber > 0 {
			start := (pageNumber - 1) * pageSize
			if start >= total {
				return pagination.Result[currencies.Currency]{
					Page:       params.Page,
					TotalCount: total,
					Items:      []currencies.Currency{},
				}, nil
			}
			end := start + pageSize
			if end > total {
				end = total
			}
			items = items[start:end]
		}

		return pagination.Result[currencies.Currency]{
			Page:       params.Page,
			TotalCount: total,
			Items:      items,
		}, nil
	})
}

func (s *Service) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	if params.Validate() != nil {
		return currencies.Currency{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate()))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.Currency, error) {
		return s.adapter.CreateCurrency(ctx, params)
	})
}

func (s *Service) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	if params.Validate() != nil {
		return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate()))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.CostBasis, error) {
		now := time.Now()
		if params.EffectiveFrom != nil && !params.EffectiveFrom.After(now) {
			return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf(
				"effective_from %s must be in the future (current time: %s)",
				params.EffectiveFrom.UTC().Format(time.RFC3339),
				now.UTC().Format(time.RFC3339),
			))
		}

		effectiveFrom := now
		if params.EffectiveFrom != nil {
			effectiveFrom = *params.EffectiveFrom
		}
		params.EffectiveFrom = &effectiveFrom

		return s.adapter.CreateCostBasis(ctx, params)
	})
}

func (s *Service) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	if params.Validate() != nil {
		return pagination.Result[currencies.CostBasis]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate()))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[currencies.CostBasis], error) {
		return s.adapter.ListCostBases(ctx, params)
	})
}
