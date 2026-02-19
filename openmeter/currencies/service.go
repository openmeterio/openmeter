package currencies

import (
	"context"

	"github.com/invopop/gobl/currency"
)

type CurrencyService interface {
	ListCurrencies(ctx context.Context) ([]Currency, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (*currency.Def, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (*CostBasis, error)
	GetCostBasesByCurrencyID(ctx context.Context, currencyID string) (CostBases, error)
}
