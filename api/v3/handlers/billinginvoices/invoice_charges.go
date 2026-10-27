package billinginvoices

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	InvoiceCustomerChargesRequest struct {
		billing.InvoicePendingLinesInput
		Options []billing.InvoicePendingLinesOption
	}
	InvoiceCustomerChargesResponse = api.InvoiceCustomerChargesResult
	InvoiceCustomerChargesHandler  = httptransport.HandlerWithArgs[InvoiceCustomerChargesRequest, InvoiceCustomerChargesResponse, string]
)

func (h *handler) InvoiceCustomerCharges() InvoiceCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID string) (InvoiceCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return InvoiceCustomerChargesRequest{}, err
			}

			var body api.InvoiceCustomerChargesInput
			if err := request.ParseOptionalBody(r, &body); err != nil {
				return InvoiceCustomerChargesRequest{}, err
			}

			input := InvoiceCustomerChargesRequest{
				InvoicePendingLinesInput: billing.InvoicePendingLinesInput{
					Customer: customer.CustomerID{
						Namespace: ns,
						ID:        customerID,
					},
					AsOf: body.AsOf,
				},
				Options: []billing.InvoicePendingLinesOption{billing.WithBypassCollectionAlignment()},
			}

			if body.ProgressiveBillingOverride != nil {
				if *body.ProgressiveBillingOverride {
					input.Options = append(input.Options, billing.WithPartialInvoiceLinesEnabled())
				} else {
					input.Options = append(input.Options, billing.WithPartialInvoiceLinesDisabled())
				}
			}

			return input, nil
		},
		func(ctx context.Context, input InvoiceCustomerChargesRequest) (InvoiceCustomerChargesResponse, error) {
			invoices, err := h.service.InvoicePendingLines(ctx, input.InvoicePendingLinesInput, input.Options...)
			if err != nil {
				return InvoiceCustomerChargesResponse{}, fmt.Errorf("collecting customer pending lines: %w", err)
			}

			out := InvoiceCustomerChargesResponse{
				Invoices: make([]api.BillingInvoice, 0, len(invoices)),
			}
			for _, invoice := range invoices {
				mappedInvoice, err := ToAPIStandardInvoice(invoice)
				if err != nil {
					return InvoiceCustomerChargesResponse{}, fmt.Errorf("mapping collected invoice: %w", err)
				}

				out.Invoices = append(out.Invoices, mappedInvoice)
			}

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[InvoiceCustomerChargesResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("invoice-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
