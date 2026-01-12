package apps

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateCustomInvoicingPaymentStatusRequest struct {
		InvoiceId api.ULID
		Body      api.BillingAppCustomInvoicingUpdatePaymentStatusRequest
	}
	UpdateCustomInvoicingPaymentStatusResponse = *struct{}
	UpdateCustomInvoicingPaymentStatusParams   = api.ULID
	UpdateCustomInvoicingPaymentStatusHandler  httptransport.HandlerWithArgs[UpdateCustomInvoicingPaymentStatusRequest, UpdateCustomInvoicingPaymentStatusResponse, UpdateCustomInvoicingPaymentStatusParams]
)

func (h *handler) UpdateCustomInvoicingPaymentStatus() UpdateCustomInvoicingPaymentStatusHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId UpdateCustomInvoicingPaymentStatusParams) (UpdateCustomInvoicingPaymentStatusRequest, error) {
			body := api.BillingAppCustomInvoicingUpdatePaymentStatusRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCustomInvoicingPaymentStatusRequest{}, err
			}

			return UpdateCustomInvoicingPaymentStatusRequest{
				InvoiceId: invoiceId,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, request UpdateCustomInvoicingPaymentStatusRequest) (UpdateCustomInvoicingPaymentStatusResponse, error) {
			return nil, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.EmptyResponseEncoder[UpdateCustomInvoicingPaymentStatusResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-custom-invoicing-payment-status"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
