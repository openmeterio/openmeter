package billinginvoices

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateBillingInvoiceRequest struct {
		Invoice billing.InvoiceID
		Update  api.UpdateInvoiceRequest
	}
	UpdateBillingInvoiceParams   = string
	UpdateBillingInvoiceResponse = api.BillingInvoice
	UpdateBillingInvoiceHandler  = httptransport.HandlerWithArgs[UpdateBillingInvoiceRequest, UpdateBillingInvoiceResponse, UpdateBillingInvoiceParams]
)

func (h *handler) UpdateBillingInvoice() UpdateBillingInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId UpdateBillingInvoiceParams) (UpdateBillingInvoiceRequest, error) {
			body := api.UpdateInvoiceRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateBillingInvoiceRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateBillingInvoiceRequest{}, err
			}

			return UpdateBillingInvoiceRequest{
				Invoice: billing.InvoiceID(models.NamespacedID{
					Namespace: ns,
					ID:        invoiceId,
				}),
				Update: body,
			}, nil
		},
		func(ctx context.Context, request UpdateBillingInvoiceRequest) (UpdateBillingInvoiceResponse, error) {
			// v3 only exposes standard invoices. UpdateStandardInvoice asserts (rather than
			// error-returns) when handed a gathering invoice, so gathering invoices must be
			// rejected here as not-found before we ever call it, mirroring how ToAPIBillingInvoice
			// rejects them on the read path.
			existing, err := h.service.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{Invoice: request.Invoice})
			if err != nil {
				return UpdateBillingInvoiceResponse{}, err
			}
			if existing.Type() != billing.InvoiceTypeStandard {
				return UpdateBillingInvoiceResponse{}, billing.NotFoundError{
					ID:     request.Invoice.ID,
					Entity: billing.EntityInvoice,
					Err:    fmt.Errorf("unsupported invoice type %q", existing.Type()),
				}
			}

			invoiceType, err := request.Update.Discriminator()
			if err != nil {
				return UpdateBillingInvoiceResponse{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "body.type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}
			switch invoiceType {
			case string(api.BillingInvoiceStandardTypeStandard):
				req, err := request.Update.AsUpdateInvoiceStandardRequest()
				if err != nil {
					return UpdateBillingInvoiceResponse{}, err
				}

				updated, err := h.service.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
					Invoice:      request.Invoice,
					ChangeSource: billing.ChangeSourceAPIRequest,
					EditFn: func(inv *billing.StandardInvoice) error {
						return mergeStandardInvoiceFromAPI(inv, req)
					},
				})
				if err != nil {
					return UpdateBillingInvoiceResponse{}, err
				}
				return ToAPIBillingInvoice(billing.NewInvoice(updated))
			default:
				err := fmt.Errorf("unsupported invoice type: %s", invoiceType)
				return UpdateBillingInvoiceResponse{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateBillingInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-invoice"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(encodeValidationIssue()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
