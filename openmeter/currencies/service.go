package currencies

import (
	"context"

	"github.com/invopop/gobl/currency"
)

type CurrencyService interface {
	ListCurrencies() ([]*currency.Def, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (*currency.Def, error)
}
