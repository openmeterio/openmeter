package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetInvoiceLineCostResponse = api.InvoiceLineCost
	GetInvoiceLineCostParams   struct {
		InvoiceID string
		LineID    string
		Params    api.GetInvoiceLineCostParams
	}
	GetInvoiceLineCostHandler httptransport.HandlerWithArgs[GetInvoiceLineCostRequest, GetInvoiceLineCostResponse, GetInvoiceLineCostParams]
)

type GetInvoiceLineCostRequest struct {
	InvoiceID billing.InvoiceID
	LineID    string
	Params    api.GetInvoiceLineCostParams
}

func (h *handler) GetInvoiceLineCost() GetInvoiceLineCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetInvoiceLineCostParams) (GetInvoiceLineCostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetInvoiceLineCostRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetInvoiceLineCostRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				LineID: params.LineID,
				Params: params.Params,
			}, nil
		},
		func(ctx context.Context, request GetInvoiceLineCostRequest) (GetInvoiceLineCostResponse, error) {
			params := cost.GetInvoiceLineCostParams{
				InvoiceID:     request.InvoiceID,
				InvoiceLineID: request.LineID,
				GroupBy:       request.Params.GroupBy,
			}

			if request.Params.WindowSize != nil {
				params.WindowSize = lo.ToPtr(meter.WindowSize(*request.Params.WindowSize))
			}

			if request.Params.WindowTimeZone != nil {
				tz, err := time.LoadLocation(*request.Params.WindowTimeZone)
				if err != nil {
					err := fmt.Errorf("invalid time zone: %w", err)
					return GetInvoiceLineCostResponse{}, models.NewGenericValidationError(err)
				}
				params.WindowTimeZone = tz
			}

			invoiceLineCost, err := h.costService.GetInvoiceLineCost(ctx, params)
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			return mapInvliceLineCostToAPI(invoiceLineCost), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetInvoiceLineCostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetInvoiceLineCost"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapInvliceLineCostToAPI(invoiceLineCost cost.InvoiceLineCost) api.InvoiceLineCost {
	// Each rows
	rows := make([]api.InvoiceLineCostRow, 0, len(invoiceLineCost.Rows))

	for _, row := range invoiceLineCost.Rows {
		rows = append(rows, mapInvliceLineCostRowToAPI(row))
	}

	response := api.InvoiceLineCost{
		From:        invoiceLineCost.From,
		To:          invoiceLineCost.To,
		Currency:    string(invoiceLineCost.Currency),
		CostPerUnit: invoiceLineCost.CostPerUnit.String(),
		Usage:       invoiceLineCost.Usage.String(),
		Cost:        invoiceLineCost.Cost.String(),
		Rows:        rows,
	}

	if invoiceLineCost.InternalCost != nil {
		response.InternalCost = lo.ToPtr(invoiceLineCost.InternalCost.String())
	}

	if invoiceLineCost.InternalCostPerUnit != nil {
		response.InternalCostPerUnit = lo.ToPtr(invoiceLineCost.InternalCostPerUnit.String())
	}

	if invoiceLineCost.Margin != nil {
		response.Margin = lo.ToPtr(invoiceLineCost.Margin.String())
	}

	if invoiceLineCost.MarginRate != nil {
		response.MarginRate = lo.ToPtr(invoiceLineCost.MarginRate.String())
	}

	return response
}

func mapInvliceLineCostRowToAPI(row cost.InvoiceLineCostRow) api.InvoiceLineCostRow {
	return api.InvoiceLineCostRow{
		WindowStart: row.WindowStart,
		WindowEnd:   row.WindowEnd,
		Usage:       row.Usage.String(),
		Cost:        row.Cost.String(),
		CostPerUnit: row.CostPerUnit.String(),
		GroupBy:     row.GroupBy,
	}
}
