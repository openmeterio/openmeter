package apps

import (
	"context"
	"fmt"
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
	UpdateCustomInvoicingPaymentStatusRequest  = appcustominvoicing.HandlePaymentTriggerInput
	UpdateCustomInvoicingPaymentStatusResponse = *struct{}
	UpdateCustomInvoicingPaymentStatusParams   = api.ULID
	UpdateCustomInvoicingPaymentStatusHandler  httptransport.HandlerWithArgs[UpdateCustomInvoicingPaymentStatusRequest, UpdateCustomInvoicingPaymentStatusResponse, UpdateCustomInvoicingPaymentStatusParams]
)

func (h *handler) UpdateCustomInvoicingPaymentStatus() UpdateCustomInvoicingPaymentStatusHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, invoiceId UpdateCustomInvoicingPaymentStatusParams) (UpdateCustomInvoicingPaymentStatusRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomInvoicingPaymentStatusRequest{}, err
			}

			body := api.BillingAppCustomInvoicingUpdatePaymentStatusRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCustomInvoicingPaymentStatusRequest{}, err
			}

			trigger, err := mapPaymentTriggerFromAPI(body.Trigger)
			if err != nil {
				return UpdateCustomInvoicingPaymentStatusRequest{}, fmt.Errorf("failed to map payment trigger: %w", err)
			}

			updatePaymentStatusRequest := UpdateCustomInvoicingPaymentStatusRequest{
				InvoiceID: billing.InvoiceID{
					ID:        invoiceId,
					Namespace: namespace,
				},
				Trigger: trigger,
			}

			if err := updatePaymentStatusRequest.Validate(); err != nil {
				return UpdateCustomInvoicingPaymentStatusRequest{}, err
			}

			return updatePaymentStatusRequest, nil
		},
		func(ctx context.Context, request UpdateCustomInvoicingPaymentStatusRequest) (UpdateCustomInvoicingPaymentStatusResponse, error) {
			_, err := h.syncService.HandlePaymentTrigger(ctx, request)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[UpdateCustomInvoicingPaymentStatusResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-custom-invoicing-payment-status"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
