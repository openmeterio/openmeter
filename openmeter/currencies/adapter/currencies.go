package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ currencies.Adapter = (*adapter)(nil)

func (a *adapter) ListCurrencies(ctx context.Context) ([]currencies.Currency, error) {
	currencyRecords, err := a.db.CustomCurrency.Query().
		Order(entdb.Asc(customcurrency.FieldCode)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}

	return lo.Map(currencyRecords, func(currency *entdb.CustomCurrency, _ int) currencies.Currency {
		return currencies.Currency{
			ID:                   currency.ID,
			Code:                 currency.Code,
			Name:                 currency.Name,
			Symbol:               currency.Symbol,
			SmallestDenomination: currency.SmallestDenomination,
			IsCustom:             true,
		}
	}), nil
}

func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (*currency.Def, error) {
	curr, err := a.db.CustomCurrency.Create().
		SetCode(params.Code).
		SetName(params.Name).
		SetSymbol(params.Symbol).
		SetSmallestDenomination(params.SmallestDenomination).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
		}
		return nil, fmt.Errorf("failed to create currency: %w", err)
	}

	return &currency.Def{
		ISOCode:              currency.Code(curr.Code),
		Name:                 curr.Name,
		Symbol:               curr.Symbol,
		SmallestDenomination: int(curr.SmallestDenomination),
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
		SetRate(alpacadecimal.NewFromFloat32(params.Rate)).
		SetEffectiveFrom(effectiveFrom).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, models.NewGenericConflictError(fmt.Errorf("failed to create cost basis: %w", err))
		}
		return nil, fmt.Errorf("failed to create cost basis: %w", err)
	}
	return &currencies.CostBasis{
		ID:            costBasis.ID,
		CurrencyID:    params.CurrencyID,
		FiatCode:      costBasis.FiatCode,
		Rate:          costBasis.Rate,
		EffectiveFrom: costBasis.EffectiveFrom,
	}, nil
}

func (a *adapter) GetCostBasesByCurrencyID(ctx context.Context, currencyID string) (currencies.CostBases, error) {
	costBases, err := a.db.CurrencyCostBasis.Query().
		Where(
			currencycostbasis.HasCurrencyWith(customcurrency.ID(currencyID)),
		).
		Order(entdb.Desc(currencycostbasis.FieldEffectiveFrom)).
		WithCurrency().
		All(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("cost basis with id: %s not found", currencyID))
		}
		return nil, fmt.Errorf("failed to get cost basis: %w", err)
	}
	return lo.Map(costBases, func(costBasis *entdb.CurrencyCostBasis, _ int) currencies.CostBasis {
		return currencies.CostBasis{
			ID:            costBasis.ID,
			CurrencyID:    costBasis.Edges.Currency.ID,
			FiatCode:      costBasis.FiatCode,
			Rate:          costBasis.Rate,
			EffectiveFrom: costBasis.EffectiveFrom,
		}
	}), nil
}
