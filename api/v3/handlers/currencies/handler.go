package currencies

import (
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
	CreateCurrency() CreateCurrencyHandler
	CreateCostBasis() CreateCostBasisHandler
	ListCostBases() ListCostBasesHandler
}

type handler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	currencyService  currencies.CurrencyService
}

func New(namespaceDecoder namespacedriver.NamespaceDecoder, currencyService currencies.CurrencyService, options ...httptransport.HandlerOption) Handler {
	return &handler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		currencyService:  currencyService,
	}
}
