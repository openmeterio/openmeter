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
	ListCurrencies(ctx context.Context, params ListCurrenciesInput) ([]Currency, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (*currency.Def, error)
}
