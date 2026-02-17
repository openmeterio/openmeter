package currencies

import (
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
	CreateCurrency() CreateCurrencyHandler
}

type handler struct {
	options         []httptransport.HandlerOption
	currencyService currencies.CurrencyService
}

func New(currencyService currencies.CurrencyService, options ...httptransport.HandlerOption) Handler {
	return &handler{
		options:         options,
		currencyService: currencyService,
	}
}
