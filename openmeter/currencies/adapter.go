package currencies

import (
	"context"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CurrenciesAdapter
	entutils.TxCreator
}

type CurrenciesAdapter interface {
	ListCurrencies(ctx context.Context) ([]Currency, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (*currency.Def, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (*CostBasis, error)
	GetCostBasesByCurrencyID(ctx context.Context, currencyID string) ([]CostBasis, error)
}
