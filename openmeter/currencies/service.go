package currencies

import (
	"context"
)

type CurrencyService interface {
	ListCurrencies(ctx context.Context) ([]Currency, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (*CostBasis, error)
	GetCostBasesByCurrencyID(ctx context.Context, currencyID string) (CostBases, error)
}
