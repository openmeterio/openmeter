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
	SubmitCustomInvoicingDraftSynchronizedRequest  = appcustominvoicing.SyncDraftInvoiceInput
	SubmitCustomInvoicingDraftSynchronizedResponse = *struct{}
	SubmitCustomInvoicingDraftSynchronizedParams   = api.ULID
	SubmitCustomInvoicingDraftSynchronizedHandler  httptransport.HandlerWithArgs[SubmitCustomInvoicingDraftSynchronizedRequest, SubmitCustomInvoicingDraftSynchronizedResponse, SubmitCustomInvoicingDraftSynchronizedParams]
)

func (h *handler) SubmitCustomInvoicingDraftSynchronized() SubmitCustomInvoicingDraftSynchronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId SubmitCustomInvoicingDraftSynchronizedParams) (SubmitCustomInvoicingDraftSynchronizedRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return SubmitCustomInvoicingDraftSynchronizedRequest{}, err
			}

			body := api.BillingAppCustomInvoicingDraftSynchronizedRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return SubmitCustomInvoicingDraftSynchronizedRequest{}, err
			}

			draftSyncRequest := SubmitCustomInvoicingDraftSynchronizedRequest{
				InvoiceID: billing.InvoiceID{
					ID:        invoiceId,
					Namespace: namespace,
				},
				UpsertInvoiceResults: mapUpsertInvoiceResultFromAPI(body.Invoicing),
			}
			if err := draftSyncRequest.Validate(); err != nil {
				return SubmitCustomInvoicingDraftSynchronizedRequest{}, err
			}

			return draftSyncRequest, nil
		},
		func(ctx context.Context, request SubmitCustomInvoicingDraftSynchronizedRequest) (SubmitCustomInvoicingDraftSynchronizedResponse, error) {
			_, err := h.syncService.SyncDraftInvoice(ctx, request)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[SubmitCustomInvoicingDraftSynchronizedResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("submit-custom-invoicing-draft-synchronized"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
