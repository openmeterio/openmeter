package apps

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	SubmitCustomInvoicingIssuingSynchronizedRequest  = appcustominvoicing.SyncIssuingInvoiceInput
	SubmitCustomInvoicingIssuingSynchronizedResponse = *struct{}
	SubmitCustomInvoicingIssuingSynchronizedParams   = api.ULID
	SubmitCustomInvoicingIssuingSynchronizedHandler  httptransport.HandlerWithArgs[SubmitCustomInvoicingIssuingSynchronizedRequest, SubmitCustomInvoicingIssuingSynchronizedResponse, SubmitCustomInvoicingIssuingSynchronizedParams]
)

func (h *handler) SubmitCustomInvoicingIssuingSynchronized() SubmitCustomInvoicingIssuingSynchronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId SubmitCustomInvoicingIssuingSynchronizedParams) (SubmitCustomInvoicingIssuingSynchronizedRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return SubmitCustomInvoicingIssuingSynchronizedRequest{}, err
			}

			body := api.BillingAppCustomInvoicingFinalizedRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return SubmitCustomInvoicingIssuingSynchronizedRequest{}, err
			}

			issuingSyncRequest := SubmitCustomInvoicingIssuingSynchronizedRequest{
				InvoiceID: billing.InvoiceID{
					ID:        invoiceId,
					Namespace: namespace,
				},
				FinalizeInvoiceResult: mapFinalizeInvoiceResultFromAPI(body),
			}

			if err := issuingSyncRequest.Validate(); err != nil {
				return SubmitCustomInvoicingIssuingSynchronizedRequest{}, err
			}

			return issuingSyncRequest, nil
		},
		func(ctx context.Context, request SubmitCustomInvoicingIssuingSynchronizedRequest) (SubmitCustomInvoicingIssuingSynchronizedResponse, error) {
			_, err := h.syncService.SyncIssuingInvoice(ctx, request)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[SubmitCustomInvoicingIssuingSynchronizedResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("submit-custom-invoicing-issuing-synchronized"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
