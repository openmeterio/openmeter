package billinginvoices

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ProgressAction string

const (
	InvoiceProgressActionApprove            ProgressAction = "approve"
	InvoiceProgressActionRetry              ProgressAction = "retry"
	InvoiceProgressActionAdvance            ProgressAction = "advance"
	InvoiceProgressActionSnapshotQuantities ProgressAction = "snapshot_quantities"
)

var (
	InvoiceProgressActions = []ProgressAction{
		InvoiceProgressActionApprove,
		InvoiceProgressActionRetry,
		InvoiceProgressActionAdvance,
		InvoiceProgressActionSnapshotQuantities,
	}
	invoiceProgressOperationNames = map[ProgressAction]string{
		InvoiceProgressActionAdvance:            "advance-invoice",
		InvoiceProgressActionApprove:            "approve-invoice",
		InvoiceProgressActionRetry:              "retry-invoice",
		InvoiceProgressActionSnapshotQuantities: "snapshot-quantities-invoice",
	}
)

type (
	ProgressInvoiceRequest struct {
		Invoice billing.InvoiceID
	}
	ProgressInvoiceResponse = api.BillingInvoice
	ProgressInvoiceParams   = string
	ProgressInvoiceHandler  httptransport.HandlerWithArgs[ProgressInvoiceRequest, ProgressInvoiceResponse, ProgressInvoiceParams]
)

func (h *handler) ProgressInvoice(action ProgressAction) ProgressInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ProgressInvoiceParams) (ProgressInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ProgressInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			if !slices.Contains(InvoiceProgressActions, action) {
				return ProgressInvoiceRequest{}, fmt.Errorf("invalid action: %s", action)
			}

			return ProgressInvoiceRequest{
				Invoice: billing.InvoiceID{
					ID:        params,
					Namespace: ns,
				},
			}, nil
		},
		func(ctx context.Context, request ProgressInvoiceRequest) (ProgressInvoiceResponse, error) {
			var invoice billing.StandardInvoice
			var err error

			switch action {
			case InvoiceProgressActionApprove:
				invoice, err = h.service.ApproveInvoice(ctx, request.Invoice)
			case InvoiceProgressActionRetry:
				invoice, err = h.service.RetryInvoice(ctx, request.Invoice)
			case InvoiceProgressActionAdvance:
				invoice, err = h.service.AdvanceInvoice(ctx, request.Invoice)
			case InvoiceProgressActionSnapshotQuantities:
				invoice, err = h.service.ForceCollectInvoice(ctx, request.Invoice)
			default:
				return ProgressInvoiceResponse{}, models.NewGenericValidationError(fmt.Errorf("invalid action: %s", action))
			}

			if err != nil {
				return ProgressInvoiceResponse{}, err
			}

			return ToAPIStandardInvoice(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[ProgressInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName(invoiceProgressOperationNames[action]),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
