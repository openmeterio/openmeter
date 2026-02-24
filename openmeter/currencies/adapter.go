package currencies

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CurrenciesAdapter
	entutils.TxCreator
}

type CurrenciesAdapter interface {
	ListCurrencies(ctx context.Context, params ListCurrenciesInput) ([]Currency, int, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (*CostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) ([]CostBasis, int, error)
}
