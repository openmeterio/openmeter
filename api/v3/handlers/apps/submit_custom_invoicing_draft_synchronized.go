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
	SubmitCustomInvoicingDraftSynchronizedRequest struct {
		InvoiceId api.ULID
		Body      api.BillingAppCustomInvoicingDraftSynchronizedRequest
	}
	SubmitCustomInvoicingDraftSynchronizedResponse = *struct{}
	SubmitCustomInvoicingDraftSynchronizedParams   = api.ULID
	SubmitCustomInvoicingDraftSynchronizedHandler  httptransport.HandlerWithArgs[SubmitCustomInvoicingDraftSynchronizedRequest, SubmitCustomInvoicingDraftSynchronizedResponse, SubmitCustomInvoicingDraftSynchronizedParams]
)

func (h *handler) SubmitCustomInvoicingDraftSynchronized() SubmitCustomInvoicingDraftSynchronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId SubmitCustomInvoicingDraftSynchronizedParams) (SubmitCustomInvoicingDraftSynchronizedRequest, error) {
			body := api.BillingAppCustomInvoicingDraftSynchronizedRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return SubmitCustomInvoicingDraftSynchronizedRequest{}, err
			}

			return SubmitCustomInvoicingDraftSynchronizedRequest{
				InvoiceId: invoiceId,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, request SubmitCustomInvoicingDraftSynchronizedRequest) (SubmitCustomInvoicingDraftSynchronizedResponse, error) {
			return nil, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.EmptyResponseEncoder[SubmitCustomInvoicingDraftSynchronizedResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("submit-custom-invoicing-draft-synchronized"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
