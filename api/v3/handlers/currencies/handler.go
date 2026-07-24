package currencies

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

var errCustomCurrenciesDisabled = errors.New("custom currencies are not enabled on this deployment of OpenMeter")

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
	CreateCurrency() CreateCurrencyHandler
	CreateCostBasis() CreateCostBasisHandler
	ListCostBases() ListCostBasesHandler
	GetCurrency() GetCurrencyHandler
}

type handler struct {
	resolveNamespace        func(ctx context.Context) (string, error)
	options                 []httptransport.HandlerOption
	service                 currencies.Service
	customCurrenciesEnabled bool
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	currencyService currencies.Service,
	customCurrenciesEnabled bool,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:        resolveNamespace,
		options:                 options,
		service:                 currencyService,
		customCurrenciesEnabled: customCurrenciesEnabled,
	}
}
