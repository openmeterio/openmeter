package currencies

import (
	"context"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CurrenciesAdapter
	entutils.TxCreator
}

type CurrenciesAdapter interface {
	ListCurrencies(ctx context.Context, params ListCurrenciesInput) ([]v3.BillingCurrency, int, error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (v3.BillingCurrencyCustom, error)
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (v3.BillingCostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) ([]v3.BillingCostBasis, int, error)
}
