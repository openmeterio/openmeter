package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type (
	GetInvoiceFeatureCostResponse = api.InvoiceFeatureCost
	GetInvoiceFeatureCostParams   struct {
		InvoiceID  string
		FeatureKey string
		Params     api.GetInvoiceFeatureCostParams
	}
	GetInvoiceFeatureCostHandler httptransport.HandlerWithArgs[GetInvoiceFeatureCostRequest, GetInvoiceFeatureCostResponse, GetInvoiceFeatureCostParams]
)

type GetInvoiceFeatureCostRequest struct {
	InvoiceID  billing.InvoiceID
	FeatureKey string
	Params     api.GetInvoiceFeatureCostParams
}

func (h *handler) GetInvoiceFeatureCost() GetInvoiceFeatureCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetInvoiceFeatureCostParams) (GetInvoiceFeatureCostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetInvoiceFeatureCostRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetInvoiceFeatureCostRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				FeatureKey: params.FeatureKey,
				Params:     params.Params,
			}, nil
		},
		func(ctx context.Context, request GetInvoiceFeatureCostRequest) (GetInvoiceFeatureCostResponse, error) {
			// Get the invoice
			invoice, err := h.service.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: request.InvoiceID,
				Expand: billing.InvoiceExpand{
					Lines:                       true,
					RecalculateGatheringInvoice: true,
				},
			})
			if err != nil {
				return GetInvoiceFeatureCostResponse{}, err
			}

			// Find the line with the feature key
			line, ok := lo.Find(invoice.Lines.OrEmpty(), func(line *billing.Line) bool {
				if line.UsageBased == nil {
					return false
				}

				return line.UsageBased.FeatureKey == request.FeatureKey
			})
			if !ok {
				return GetInvoiceFeatureCostResponse{}, models.NewGenericNotFoundError(
					fmt.Errorf("feature not found in invoice: %s", request.FeatureKey),
				)
			}

			if line.UsageBased == nil {
				return GetInvoiceFeatureCostResponse{}, models.NewGenericConflictError(
					fmt.Errorf("invoice line is not usage based for feature: %s", request.FeatureKey),
				)
			}

			// Get the feature
			feature, err := h.featureService.GetFeature(ctx, request.InvoiceID.Namespace, request.FeatureKey, false)
			if err != nil {
				return GetInvoiceFeatureCostResponse{}, err
			}

			if feature.MeterSlug == nil {
				return GetInvoiceFeatureCostResponse{}, models.NewGenericConflictError(
					fmt.Errorf("feature has no meter: %s", request.FeatureKey),
				)
			}

			// Get the meter
			met, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.InvoiceID.Namespace,
				IDOrSlug:  *feature.MeterSlug,
			})
			if err != nil {
				return GetInvoiceFeatureCostResponse{}, err
			}

			// Convert the feature's meter group by filters to a map of filter group by
			meterGroupByFilters := make(map[string][]string)
			for k, v := range feature.MeterGroupByFilters {
				meterGroupByFilters[k] = []string{v}
			}

			meterQueryParams := streaming.QueryParams{
				From:          &line.Period.Start,
				To:            &line.Period.End,
				FilterGroupBy: meterGroupByFilters,
			}

			if request.Params.GroupBy != nil {
				meterQueryParams.GroupBy = *request.Params.GroupBy
			}

			if request.Params.WindowSize != nil {
				meterQueryParams.WindowSize = lo.ToPtr(meter.WindowSize(*request.Params.WindowSize))
			}

			if request.Params.WindowTimeZone != nil {
				tz, err := time.LoadLocation(*request.Params.WindowTimeZone)
				if err != nil {
					err := fmt.Errorf("invalid time zone: %w", err)
					return GetInvoiceFeatureCostResponse{}, models.NewGenericValidationError(err)
				}
				meterQueryParams.WindowTimeZone = tz
			}

			// Get usage for the line
			usageRows, err := h.streamingService.QueryMeter(ctx, request.InvoiceID.Namespace, met, meterQueryParams)
			if err != nil {
				return GetInvoiceFeatureCostResponse{}, err
			}

			// Get the cost per unit
			costPerUnit := alpacadecimal.NewFromInt(0)

			if !line.UsageBased.Quantity.IsZero() {
				costPerUnit = line.Totals.Amount.Div(*line.UsageBased.Quantity)
			}

			// Calculate the cost for each window
			rows := make([]api.InvoiceFeatureCostRow, 0, len(usageRows))

			for _, row := range usageRows {
				usage := alpacadecimal.NewFromFloat(row.Value)
				cost := usage.Mul(costPerUnit)

				rows = append(rows, api.InvoiceFeatureCostRow{
					WindowStart: row.WindowStart,
					WindowEnd:   row.WindowEnd,
					Usage:       usage.String(),
					Cost:        cost.String(),
					GroupBy:     row.GroupBy,
				})
			}

			return api.InvoiceFeatureCost{
				From:        line.Period.Start,
				To:          line.Period.End,
				Currency:    string(line.Currency),
				CostPerUnit: costPerUnit.String(),
				TotalUsage:  line.UsageBased.Quantity.String(),
				TotalCost:   line.Totals.Amount.String(),
				Rows:        rows,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetInvoiceFeatureCostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetInvoiceFeatureCost"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
