package productcatalog

import (
	"context"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// ResolvedCurrency identifies the managed currency behind a catalog currency
// code. Fiat currencies have no managed resource ID.
type ResolvedCurrency struct {
	ID   string
	Code currency.Code
	Type currencyx.CurrencyType
}

type CurrencyResolver interface {
	Resolve(ctx context.Context, namespace string, code currency.Code) (ResolvedCurrency, error)
	HasCostBasis(ctx context.Context, namespace string, customCurrency ResolvedCurrency, fiatCurrency currency.Code) (bool, error)
}
