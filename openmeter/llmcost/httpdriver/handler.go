package httpdriver

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListPrices() ListPricesHandler
	GetPrice() GetPriceHandler
	ResolvePrice() ResolvePriceHandler
	ListOverrides() ListOverridesHandler
	CreateOverride() CreateOverrideHandler
	UpdateOverride() UpdateOverrideHandler
	DeleteOverride() DeleteOverrideHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          llmcost.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service llmcost.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
