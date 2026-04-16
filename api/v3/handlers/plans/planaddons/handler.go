package planaddons

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListPlanAddons() ListPlanAddonsHandler
	GetPlanAddon() GetPlanAddonHandler
	CreatePlanAddon() CreatePlanAddonHandler
	UpdatePlanAddon() UpdatePlanAddonHandler
	DeletePlanAddon() DeletePlanAddonHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          plan.Service
	addonService     planaddon.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service plan.Service,
	addonService planaddon.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		addonService:     addonService,
		options:          options,
	}
}
