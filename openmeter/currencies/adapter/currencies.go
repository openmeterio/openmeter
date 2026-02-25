package adapter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasiseffectivefrom"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ currencies.Adapter = (*adapter)(nil)

func (a *adapter) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]currencies.Currency, int, error) {
	var all []currencies.Currency

	includeCustom := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeCustom
	includeFiat := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeFiat

	if includeCustom {
		currencyRecords, err := a.db.CustomCurrency.Query().
			Order(entdb.Asc(customcurrency.FieldCode)).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list currencies: %w", err)
		}

		all = append(all, lo.Map(currencyRecords, func(c *entdb.CustomCurrency, _ int) currencies.Currency {
			return currencies.Currency{
				ID:       c.ID,
				Code:     c.Code,
				Name:     c.Name,
				Symbol:   c.Symbol,
				IsCustom: true,
			}
		})...)
	}

	if includeFiat {
		fiat := lo.Map(lo.Filter(
			currency.Definitions(),
			func(def *currency.Def, _ int) bool {
				// NOTE: this filters out non-iso currencies such as crypto
				return def.ISONumeric != ""
			},
		), func(def *currency.Def, _ int) currencies.Currency {
			return currencies.Currency{
				Code:     def.ISOCode.String(),
				Name:     def.Name,
				Symbol:   def.Symbol,
				IsCustom: false,
			}
		})
		all = append(all, fiat...)
	}

	total := len(all)

	// Apply page-based pagination
	pageSize := params.Page.PageSize
	pageNumber := params.Page.PageNumber
	if pageSize > 0 && pageNumber > 0 {
		start := (pageNumber - 1) * pageSize
		if start >= total {
			return []currencies.Currency{}, total, nil
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		all = all[start:end]
	}

	return all, total, nil
}

func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	curr, err := a.db.CustomCurrency.Create().
		SetCode(params.Code).
		SetName(params.Name).
		SetSymbol(params.Symbol).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
		}
		return currencies.Currency{}, fmt.Errorf("failed to create currency: %w", err)
	}

	return currencies.Currency{
		ID:       curr.ID,
		Code:     curr.Code,
		Name:     curr.Name,
		Symbol:   curr.Symbol,
		IsCustom: true,
	}, nil
}

func (a *adapter) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (*currencies.CostBasis, error) {
	effectiveFrom := time.Now()
	if params.EffectiveFrom != nil {
		if params.EffectiveFrom.Before(time.Now()) {
			return nil, models.NewGenericConflictError(fmt.Errorf("effective from must be in the future"))
		}
		effectiveFrom = *params.EffectiveFrom
	}

	costBasis, err := a.db.CurrencyCostBasis.Create().
		SetCurrencyID(params.CurrencyID).
		SetFiatCode(params.FiatCode).
		SetRate(params.Rate).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, models.NewGenericConflictError(fmt.Errorf("failed to create cost basis: %w", err))
		}
		return nil, fmt.Errorf("failed to create cost basis: %w", err)
	}

	_, err = a.db.CurrencyCostBasisEffectiveFrom.Create().
		SetCostBasisID(costBasis.ID).
		SetEffectiveFrom(effectiveFrom).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost basis effective from: %w", err)
	}

	return &currencies.CostBasis{
		ID:            costBasis.ID,
		CurrencyID:    params.CurrencyID,
		FiatCode:      costBasis.FiatCode,
		Rate:          costBasis.Rate,
		EffectiveFrom: effectiveFrom,
	}, nil
}

func (a *adapter) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) ([]currencies.CostBasis, int, error) {
	q := a.db.CurrencyCostBasis.Query().
		Where(currencycostbasis.HasCurrencyWith(customcurrency.ID(params.CurrencyID)))

	if params.FilterFiatCode != nil {
		q = q.Where(currencycostbasis.FiatCode(*params.FilterFiatCode))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count cost bases: %w", err)
	}

	q = q.WithEffectiveFromHistory(func(q *entdb.CurrencyCostBasisEffectiveFromQuery) {
		q.Order(entdb.Desc(currencycostbasiseffectivefrom.FieldEffectiveFrom))
	})

	if params.Page.PageSize > 0 && params.Page.PageNumber > 0 {
		q = q.Offset((params.Page.PageNumber - 1) * params.Page.PageSize).Limit(params.Page.PageSize)
	}

	costBases, err := q.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list cost bases: %w", err)
	}

	sort.Slice(costBases, func(i, j int) bool {
		var ti, tj time.Time
		if len(costBases[i].Edges.EffectiveFromHistory) > 0 {
			ti = costBases[i].Edges.EffectiveFromHistory[0].EffectiveFrom
		}
		if len(costBases[j].Edges.EffectiveFromHistory) > 0 {
			tj = costBases[j].Edges.EffectiveFromHistory[0].EffectiveFrom
		}
		return ti.After(tj)
	})

	return lo.Map(costBases, func(costBasis *entdb.CurrencyCostBasis, _ int) currencies.CostBasis {
		var effectiveFrom time.Time
		if len(costBasis.Edges.EffectiveFromHistory) > 0 {
			effectiveFrom = costBasis.Edges.EffectiveFromHistory[0].EffectiveFrom
		}
		return currencies.CostBasis{
			ID:            costBasis.ID,
			CurrencyID:    params.CurrencyID,
			FiatCode:      costBasis.FiatCode,
			Rate:          costBasis.Rate,
			EffectiveFrom: effectiveFrom,
		}
	}), total, nil
}
