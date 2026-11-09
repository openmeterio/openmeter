package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AppHandler
}

type AppHandler interface {
	DraftSyncronized() DraftSyncronizedHandler
	IssuingSyncronized() IssuingSyncronizedHandler
	UpdatePaymentStatus() UpdatePaymentStatusHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service         appcustominvoicing.SyncService
	taxCodeResolver productcatalog.TaxCodeResolver

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
	service appcustominvoicing.SyncService,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	taxCodeResolver productcatalog.TaxCodeResolver,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		service:          service,
		taxCodeResolver:  taxCodeResolver,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}
}

func (h *handler) mapInvoiceToAPI(ctx context.Context, invoice billing.StandardInvoice) (api.Invoice, error) {
	taxCodes, err := billinghttpdriver.ResolveWorkflowTaxCodes(ctx, h.taxCodeResolver, invoice.Namespace, invoice.Workflow.Config)
	if err != nil {
		return api.Invoice{}, err
	}

	return billinghttpdriver.MapStandardInvoiceToAPI(invoice, taxCodes)
}
