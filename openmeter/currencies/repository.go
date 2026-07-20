package currencies

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Repository interface {
	entutils.TxCreator

	CurrencyRepository
	CostBasisRepository
}

type CurrencyRepository interface {
	ListCustomCurrencies(ctx context.Context, params ListCurrenciesInput) (pagination.Result[Currency], error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	GetCurrency(ctx context.Context, params GetCurrencyInput) (Currency, error)
}

type CostBasisRepository interface {
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (CostBasis, error)
	GetCostBasis(ctx context.Context, params GetCostBasisInput) (CostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) (pagination.Result[CostBasis], error)
}
