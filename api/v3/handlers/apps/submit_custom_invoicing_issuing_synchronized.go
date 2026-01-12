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
	SubmitCustomInvoicingIssuingSynchronizedRequest struct {
		InvoiceId api.ULID
		Body      api.BillingAppCustomInvoicingFinalizedRequest
	}
	SubmitCustomInvoicingIssuingSynchronizedResponse = *struct{}
	SubmitCustomInvoicingIssuingSynchronizedParams   = api.ULID
	SubmitCustomInvoicingIssuingSynchronizedHandler  httptransport.HandlerWithArgs[SubmitCustomInvoicingIssuingSynchronizedRequest, SubmitCustomInvoicingIssuingSynchronizedResponse, SubmitCustomInvoicingIssuingSynchronizedParams]
)

func (h *handler) SubmitCustomInvoicingIssuingSynchronized() SubmitCustomInvoicingIssuingSynchronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId SubmitCustomInvoicingIssuingSynchronizedParams) (SubmitCustomInvoicingIssuingSynchronizedRequest, error) {
			body := api.BillingAppCustomInvoicingFinalizedRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return SubmitCustomInvoicingIssuingSynchronizedRequest{}, err
			}

			return SubmitCustomInvoicingIssuingSynchronizedRequest{
				InvoiceId: invoiceId,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, request SubmitCustomInvoicingIssuingSynchronizedRequest) (SubmitCustomInvoicingIssuingSynchronizedResponse, error) {
			return nil, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.EmptyResponseEncoder[SubmitCustomInvoicingIssuingSynchronizedResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("submit-custom-invoicing-issuing-synchronized"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
