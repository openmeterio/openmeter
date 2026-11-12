package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AddonHandler
}

type AddonHandler interface {
	ListAddons() ListAddonsHandler
	CreateAddon() CreateAddonHandler
	DeleteAddon() DeleteAddonHandler
	GetAddon() GetAddonHandler
	UpdateAddon() UpdateAddonHandler
	PublishAddon() PublishAddonHandler
	ArchiveAddon() ArchiveAddonHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          addon.Service
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
	service addon.Service,
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

// taxCodesForAddons looks up, in one batch, the Tax Codes referenced by the add-ons' rate cards,
// so v1 responses can show their stripe.code.
func (h *handler) taxCodesForAddons(ctx context.Context, namespace string, addons ...addon.Addon) (map[string]*taxcode.TaxCode, error) {
	var rateCards productcatalog.RateCards

	for _, a := range addons {
		rateCards = append(rateCards, a.RateCards.AsProductCatalogRateCards()...)
	}

	taxCodes, err := h.taxCodeResolver.BatchResolve(ctx, namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
