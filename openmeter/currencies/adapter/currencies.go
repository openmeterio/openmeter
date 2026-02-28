package adapter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasiseffectivefrom"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ currencies.Adapter = (*adapter)(nil)

func (a *adapter) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]v3.BillingCurrency, int, error) {
	var all []v3.BillingCurrency

	includeCustom := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeCustom
	includeFiat := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeFiat

	if includeCustom {
		currencyRecords, err := a.db.CustomCurrency.Query().
			Order(entdb.Asc(customcurrency.FieldCode)).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list currencies: %w", err)
		}

		for _, c := range currencyRecords {
			var item v3.BillingCurrency
			if err := item.FromBillingCurrencyCustom(v3.BillingCurrencyCustom{
				Id:        c.ID,
				Code:      c.Code,
				Name:      c.Name,
				Symbol:    &c.Symbol,
				Type:      v3.BillingCurrencyCustomTypeCustom,
				CreatedAt: &c.CreatedAt,
			}); err != nil {
				return nil, 0, fmt.Errorf("failed to construct BillingCurrencyCustom: %w", err)
			}
			all = append(all, item)
		}
	}

	if includeFiat {
		for _, def := range lo.Filter(currency.Definitions(), func(def *currency.Def, _ int) bool {
			// NOTE: this filters out non-iso currencies such as crypto
			return def.ISONumeric != ""
		}) {
			var item v3.BillingCurrency
			if err := item.FromBillingCurrencyFiat(v3.BillingCurrencyFiat{
				Code:   def.ISOCode.String(),
				Name:   def.Name,
				Symbol: &def.Symbol,
				Type:   v3.BillingCurrencyFiatTypeFiat,
			}); err != nil {
				return nil, 0, fmt.Errorf("failed to construct BillingCurrencyFiat: %w", err)
			}
			all = append(all, item)
		}
	}

	total := len(all)

	// Apply page-based pagination
	pageSize := params.Page.PageSize
	pageNumber := params.Page.PageNumber
	if pageSize > 0 && pageNumber > 0 {
		start := (pageNumber - 1) * pageSize
		if start >= total {
			return []v3.BillingCurrency{}, total, nil
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		all = all[start:end]
	}

	return all, total, nil
}

func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (v3.BillingCurrencyCustom, error) {
	curr, err := a.db.CustomCurrency.Create().
		SetCode(params.Code).
		SetName(params.Name).
		SetSymbol(params.Symbol).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return v3.BillingCurrencyCustom{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
		}
		return v3.BillingCurrencyCustom{}, fmt.Errorf("failed to create currency: %w", err)
	}

	return v3.BillingCurrencyCustom{
		Id:        curr.ID,
		Code:      curr.Code,
		Name:      curr.Name,
		Symbol:    &curr.Symbol,
		Type:      "custom",
		CreatedAt: &curr.CreatedAt,
	}, nil
}

func (a *adapter) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (v3.BillingCostBasis, error) {
	effectiveFrom := time.Now()
	if params.EffectiveFrom != nil {
		now := time.Now()
		if !params.EffectiveFrom.After(now) {
			return v3.BillingCostBasis{}, models.NewGenericValidationError(fmt.Errorf(
				"effective_from %s must be in the future (current time: %s)",
				params.EffectiveFrom.UTC().Format(time.RFC3339),
				now.UTC().Format(time.RFC3339),
			))
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
			return v3.BillingCostBasis{}, models.NewGenericConflictError(fmt.Errorf("failed to create cost basis: %w", err))
		}
		return v3.BillingCostBasis{}, fmt.Errorf("failed to create cost basis: %w", err)
	}

	_, err = a.db.CurrencyCostBasisEffectiveFrom.Create().
		SetCostBasisID(costBasis.ID).
		SetEffectiveFrom(effectiveFrom).
		Save(ctx)
	if err != nil {
		return v3.BillingCostBasis{}, fmt.Errorf("failed to create cost basis effective from: %w", err)
	}

	return v3.BillingCostBasis{
		Id:            costBasis.ID,
		CurrencyId:    params.CurrencyID,
		FiatCode:      costBasis.FiatCode,
		Rate:          costBasis.Rate.String(),
		EffectiveFrom: &effectiveFrom,
		CreatedAt:     &costBasis.CreatedAt,
	}, nil
}

func (a *adapter) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) ([]v3.BillingCostBasis, int, error) {
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

	return lo.Map(costBases, func(costBasis *entdb.CurrencyCostBasis, _ int) v3.BillingCostBasis {
		var effectiveFrom *time.Time
		if len(costBasis.Edges.EffectiveFromHistory) > 0 {
			t := costBasis.Edges.EffectiveFromHistory[0].EffectiveFrom
			effectiveFrom = &t
		}
		return v3.BillingCostBasis{
			Id:            costBasis.ID,
			CurrencyId:    params.CurrencyID,
			FiatCode:      costBasis.FiatCode,
			Rate:          costBasis.Rate.String(),
			EffectiveFrom: effectiveFrom,
			CreatedAt:     &costBasis.CreatedAt,
		}
	}), total, nil
}
