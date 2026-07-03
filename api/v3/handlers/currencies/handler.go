package currencies

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
	CreateCurrency() CreateCurrencyHandler
	CreateCostBasis() CreateCostBasisHandler
	ListCostBases() ListCostBasesHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	options          []httptransport.HandlerOption
	service          currencies.Service
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	currencyService currencies.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		options:          options,
		service:          currencyService,
	}
}
