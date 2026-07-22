package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ currencies.Service = (*service)(nil)

type service struct {
	adapter currencies.Repository
}

func New(repo currencies.Repository) (currencies.Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("currencies repository is required")
	}

	return &service{
		adapter: repo,
	}, nil
}

func (s *service) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[currencies.Currency]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[currencies.Currency], error) {
		includeCustom := params.CurrencyType == nil || *params.CurrencyType == currencies.CurrencyTypeCustom
		includeFiat := params.CurrencyType == nil || *params.CurrencyType == currencies.CurrencyTypeFiat

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
			filteredMatchCode, err := lo.FilterErr(currency.Definitions(), func(def *currency.Def, _ int) (bool, error) {
				// NOTE: this filters out non-iso currencies such as crypto
				if def.ISONumeric == "" {
					return false, nil
				}

				return matchesCurrencyFilters(params, "", def.ISOCode.String())
			})
			if err != nil {
				return pagination.Result[currencies.Currency]{}, fmt.Errorf("filtering fiat currencies by code: %w", err)
			}

			for _, def := range filteredMatchCode {
				curr, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
					WithCode(currencyx.Code(def.ISOCode)).
					Build()
				if err != nil {
					return pagination.Result[currencies.Currency]{}, fmt.Errorf("failed to create FIAT currency with code [%s]: %w", def.ISOCode, err)
				}

				items = append(items, currencies.Currency{
					Currency: curr,
				})
			}
		}

		slices.SortFunc(items, func(a, b currencies.Currency) int {
			result := 0

			if params.OrderBy == currencies.OrderByName {
				result = strings.Compare(a.Details().Name, b.Details().Name)
			} else {
				result = strings.Compare(a.Details().Code.String(), b.Details().Code.String())
			}

			if params.Order == sortx.OrderDesc {
				return -result
			}

			return result
		})

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

func matchesCurrencyFilters(params currencies.ListCurrenciesInput, id, code string) (bool, error) {
	hasIDFilter := params.ID != nil && !params.ID.IsEmpty()
	hasCodeFilter := params.Code != nil && !params.Code.IsEmpty()

	if !hasIDFilter && !hasCodeFilter {
		return true, nil
	}

	idMatches := false
	if hasIDFilter {
		var err error
		idMatches, err = params.ID.Match(id)
		if err != nil {
			return false, fmt.Errorf("matching currency id: %w", err)
		}
	}

	codeMatches := false
	if hasCodeFilter {
		var err error
		codeMatches, err = params.Code.Match(code)
		if err != nil {
			return false, fmt.Errorf("matching currency code: %w", err)
		}
	}

	if params.Union {
		return idMatches || codeMatches, nil
	}

	return (!hasIDFilter || idMatches) && (!hasCodeFilter || codeMatches), nil
}

func (s *service) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	if err := params.Validate(); err != nil {
		return currencies.Currency{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.Currency, error) {
		return s.adapter.CreateCurrency(ctx, params)
	})
}

func (s *service) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	if err := params.Validate(); err != nil {
		return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.CostBasis, error) {
		now := clock.Now()

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

		if params.EffectiveTo != nil && !effectiveFrom.Before(*params.EffectiveTo) {
			return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf(
				"effective_to %s must be after effective_from %s",
				params.EffectiveTo.UTC().Format(time.RFC3339),
				effectiveFrom.UTC().Format(time.RFC3339),
			))
		}

		return s.adapter.CreateCostBasis(ctx, params)
	})
}

func (s *service) GetCostBasis(ctx context.Context, params currencies.GetCostBasisInput) (currencies.CostBasis, error) {
	if err := params.Validate(); err != nil {
		return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.CostBasis, error) {
		return s.adapter.GetCostBasis(ctx, params)
	})
}

func (s *service) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[currencies.CostBasis]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[currencies.CostBasis], error) {
		return s.adapter.ListCostBases(ctx, params)
	})
}

func (s *service) GetCurrency(ctx context.Context, params currencies.GetCurrencyInput) (currencies.Currency, error) {
	if err := params.Validate(); err != nil {
		return currencies.Currency{}, fmt.Errorf("invalid input parameters: %w", err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.Currency, error) {
		return s.adapter.GetCurrency(ctx, params)
	})
}

func (s *service) GetCostBasisAt(ctx context.Context, params currencies.GetCostBasisAtInput) (currencies.CostBasis, error) {
	if err := params.Validate(); err != nil {
		return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", err))
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.CostBasis, error) {
		return s.adapter.GetCostBasisAt(ctx, params)
	})
}
