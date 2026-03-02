package currencies

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	CurrenciesAdapter
	entutils.TxCreator
}

type CurrenciesAdapter interface {
	ListCustomCurrencies(ctx context.Context, params ListCurrenciesInput) (pagination.Result[Currency], error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (CostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) (pagination.Result[CostBasis], error)
}
