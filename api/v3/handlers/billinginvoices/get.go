package billinginvoices

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetBillingInvoiceRequest  = billing.GetInvoiceByIdInput
	GetBillingInvoiceResponse = api.BillingInvoice
	GetBillingInvoiceParams   = string
	GetBillingInvoiceHandler  = httptransport.HandlerWithArgs[GetBillingInvoiceRequest, GetBillingInvoiceResponse, GetBillingInvoiceParams]
)

func (h *handler) GetBillingInvoice() GetBillingInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId GetBillingInvoiceParams) (GetBillingInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetBillingInvoiceRequest{}, err
			}

			return GetBillingInvoiceRequest{
				Invoice: billing.InvoiceID(models.NamespacedID{
					Namespace: ns,
					ID:        invoiceId,
				}),
				Expand: billing.InvoiceExpandAll,
			}, nil
		},
		func(ctx context.Context, request GetBillingInvoiceRequest) (GetBillingInvoiceResponse, error) {
			inv, err := h.service.GetInvoiceById(ctx, request)
			if err != nil {
				return GetBillingInvoiceResponse{}, err
			}

			return ToAPIBillingInvoice(inv)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetBillingInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-billing-invoice"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
