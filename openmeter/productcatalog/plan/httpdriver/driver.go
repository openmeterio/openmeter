package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	PlanHandler
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

var _ Handler = (*handler)(nil)

type handler struct {
	service          plan.Service
	namespaceDecoder namespacedriver.NamespaceDecoder
	taxCodeResolver  productcatalog.TransactionalTaxCodeResolver
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
	taxCodeResolver productcatalog.TransactionalTaxCodeResolver,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		taxCodeResolver:  taxCodeResolver,
		options:          options,
	}
}

// taxCodesForPlans looks up, in one batch, the Tax Codes referenced by the plans' rate cards, so
// v1 responses can show their stripe.code.
func (h *handler) taxCodesForPlans(ctx context.Context, namespace string, plans ...plan.Plan) (map[string]*taxcode.TaxCode, error) {
	var rateCards productcatalog.RateCards

	for _, p := range plans {
		for _, phase := range p.Phases {
			rateCards = append(rateCards, phase.RateCards...)
		}
	}

	taxCodes, err := h.taxCodeResolver.BatchResolve(ctx, namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
