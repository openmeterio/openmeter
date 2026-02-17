package adapter

import (
	"context"
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

var _ currencies.Adapter = (*adapter)(nil)

func (a *adapter) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]currencies.Currency, error) {
	currencyRecords, err := a.db.CustomCurrency.Query().
		Order(entdb.Asc(customcurrency.FieldCode)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}

	return lo.Map(currencyRecords, func(currency *entdb.CustomCurrency, _ int) currencies.Currency {
		return currencies.Currency{
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
