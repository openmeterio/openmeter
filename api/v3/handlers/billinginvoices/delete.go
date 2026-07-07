package billinginvoices

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteBillingInvoiceRequest struct {
		Invoice billing.InvoiceID
	}
	DeleteBillingInvoiceParams   = string
	DeleteBillingInvoiceResponse = any
	DeleteBillingInvoiceHandler  = httptransport.HandlerWithArgs[DeleteBillingInvoiceRequest, DeleteBillingInvoiceResponse, DeleteBillingInvoiceParams]
)

func (h *handler) DeleteBillingInvoice() DeleteBillingInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId DeleteBillingInvoiceParams) (DeleteBillingInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteBillingInvoiceRequest{}, err
			}

			return DeleteBillingInvoiceRequest{
				Invoice: billing.InvoiceID(models.NamespacedID{
					Namespace: ns,
					ID:        invoiceId,
				}),
			}, nil
		},
		func(ctx context.Context, request DeleteBillingInvoiceRequest) (DeleteBillingInvoiceResponse, error) {
			existing, err := h.service.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: request.Invoice,
				Expand:  billing.InvoiceExpandAll,
			})
			if err != nil {
				return nil, err
			}
			if existing.Type() != billing.InvoiceTypeStandard {
				return nil, billing.NotFoundError{
					ID:     request.Invoice.ID,
					Entity: billing.EntityInvoice,
					Err:    fmt.Errorf("unsupported invoice type %q", existing.Type()),
				}
			}

			if err := billing.ValidateAPIInvoiceDeleteSupported(existing); err != nil {
				return nil, err
			}

			invoice, err := h.service.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
				Invoice:        request.Invoice,
				DeletionSource: billing.ChangeSourceAPIRequest,
			})
			if err != nil {
				return nil, err
			}

			// Given we are doing background processing, we might be in any delete.* state, but in case we ended up in delete.failed let's have
			// proper return code for the API (otherwise we would return 200)
			if invoice.Status == billing.StandardInvoiceStatusDeleteFailed {
				// If we have validation issues we return them as the deletion sync handler
				// yields validation errors
				if len(invoice.ValidationIssues) > 0 {
					return nil, billing.ValidationError{
						Err: invoice.ValidationIssues.AsError(),
					}
				}

				return nil, billing.ValidationError{
					Err: fmt.Errorf("%w [status=%s]", billing.ErrInvoiceDeleteFailed, invoice.Status),
				}
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteBillingInvoiceResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-invoice"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
