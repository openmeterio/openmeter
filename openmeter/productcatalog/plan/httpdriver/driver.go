package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	PlanHandler
	PlanPhaseHandler
}

type PlanHandler interface {
	ListPlans() ListPlansHandler
	CreatePlan() CreatePlanHandler
	DeletePlan() DeletePlanHandler
	GetPlan() GetPlanHandler
	UpdatePlan() UpdatePlanHandler
	NextPlan() NextPlanHandler
	PublishPlan() PublishPlanHandler
	ArchivePlan() ArchivePlanHandler
}

type PlanPhaseHandler interface {
	ListPhases() ListPhasesHandler
	CreatePhase() CreatePhaseHandler
	DeletePhase() DeletePhaseHandler
	GetPhase() GetPhaseHandler
	UpdatePhase() UpdatePhaseHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          plan.Service
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service plan.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}
}
