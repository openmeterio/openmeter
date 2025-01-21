package httpdriver

import "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
}

type handler struct {
	options []httptransport.HandlerOption
}

func New(options ...httptransport.HandlerOption) Handler {
	return &handler{
		options: options,
	}
}
