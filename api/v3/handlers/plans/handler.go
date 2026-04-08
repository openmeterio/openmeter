package plans

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	GetPlan() GetPlanHandler
	CreatePlan() CreatePlanHandler
	UpdatePlan() UpdatePlanHandler
	DeletePlan() DeletePlanHandler
	PublishPlan() PublishPlanHandler
	ArchivePlan() ArchivePlanHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          plan.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service plan.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
