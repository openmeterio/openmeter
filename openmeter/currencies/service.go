package currencies

import (
	"context"
)

type CurrencyService interface {
	ListCurrencies(ctx context.Context, params ListCurrenciesInput) ([]Currency, int, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (*CostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) ([]CostBasis, int, error)
}
