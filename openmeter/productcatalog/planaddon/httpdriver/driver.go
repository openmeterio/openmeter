package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	PlanAddonHandler
}

type PlanAddonHandler interface {
	ListPlanAddons() ListPlanAddonsHandler
	CreatePlanAddon() CreatePlanAddonHandler
	DeletePlanAddon() DeletePlanAddonHandler
	GetPlanAddon() GetPlanAddonHandler
	UpdatePlanAddon() UpdatePlanAddonHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          planaddon.Service
	namespaceDecoder namespacedriver.NamespaceDecoder
	taxCodeResolver  productcatalog.TaxCodeResolver
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
	service planaddon.Service,
	// Read-only: plan add-on endpoints only reference an add-on by ID and never write rate cards.
	taxCodeResolver productcatalog.TaxCodeResolver,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		taxCodeResolver:  taxCodeResolver,
		options:          options,
	}
}

// taxCodesForPlanAddons looks up, in one batch, the Tax Codes referenced by the rendered add-ons'
// rate cards, so v1 responses can show their stripe.code.
func (h *handler) taxCodesForPlanAddons(ctx context.Context, namespace string, planAddons ...planaddon.PlanAddon) (map[string]*taxcode.TaxCode, error) {
	var rateCards productcatalog.RateCards

	for _, pa := range planAddons {
		rateCards = append(rateCards, pa.Addon.RateCards.AsProductCatalogRateCards()...)
	}

	taxCodes, err := h.taxCodeResolver.BatchResolve(ctx, namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
