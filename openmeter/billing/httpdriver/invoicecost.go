package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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
			// Get the invoice
			invoice, err := h.service.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
				Invoice: request.InvoiceID,
				Expand: billing.InvoiceExpand{
					Lines:                       true,
					RecalculateGatheringInvoice: true,
				},
			})
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			// Find the line with the feature key
			line, ok := lo.Find(invoice.Lines.OrEmpty(), func(line *billing.Line) bool {
				if line.UsageBased == nil {
					return false
				}

				return line.ID == request.LineID
			})
			if !ok {
				return GetInvoiceLineCostResponse{}, models.NewGenericNotFoundError(
					fmt.Errorf("line not found in invoice: %s", request.LineID),
				)
			}

			if line.UsageBased == nil {
				return GetInvoiceLineCostResponse{}, models.NewGenericConflictError(
					fmt.Errorf("not a usage based line: %s", request.LineID),
				)
			}

			// Get the feature
			feature, err := h.featureService.GetFeature(ctx, request.InvoiceID.Namespace, line.UsageBased.FeatureKey, false)
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			if feature.MeterSlug == nil {
				return GetInvoiceLineCostResponse{}, models.NewGenericConflictError(
					fmt.Errorf("no meter for feature: %s", line.UsageBased.FeatureKey),
				)
			}

			// Get the meter
			met, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.InvoiceID.Namespace,
				IDOrSlug:  *feature.MeterSlug,
			})
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			// Get the customer
			customer, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: lo.ToPtr(invoice.CustomerID()),
			})
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			if customer == nil {
				return GetInvoiceLineCostResponse{}, fmt.Errorf("customer cannot be nil")
			}

			// Query the meter
			meterQueryParams := streaming.QueryParams{
				From:           &line.Period.Start,
				To:             &line.Period.End,
				FilterGroupBy:  feature.MeterGroupByFilters,
				FilterCustomer: []streaming.Customer{*customer},
				// We ignore late events because the data is ingested after the invoice is collected
				IgnoreLateEvents: invoice.CollectionAt,
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
					return GetInvoiceLineCostResponse{}, models.NewGenericValidationError(err)
				}
				meterQueryParams.WindowTimeZone = tz
			}

			// Get usage for the line
			usageRows, err := h.streamingService.QueryMeter(ctx, request.InvoiceID.Namespace, met, meterQueryParams)
			if err != nil {
				return GetInvoiceLineCostResponse{}, err
			}

			// Get the cost per unit
			costPerUnit := alpacadecimal.NewFromInt(0)

			if !line.UsageBased.Quantity.IsZero() {
				costPerUnit = line.Totals.Amount.Div(*line.UsageBased.Quantity)
			}

			// TODO: This should be a dependency of the handler
			// Get token cost
			modelDev := ModelCostProvider{}
			if err := modelDev.initialize(); err != nil {
				return GetInvoiceLineCostResponse{}, fmt.Errorf("failed to initialize model dev: %w", err)
			}

			totalInternalCost := alpacadecimal.NewFromInt(0)
			internalCostPerUnit := alpacadecimal.NewFromInt(0)

			// TODO: store cost on the feature
			// If the row has a provider, model, and type, we can calculate the internal cost per unit
			providerFilter, hasProviderFilter := feature.MeterGroupByFilters["provider"]
			modelFilter, hasModelFilter := feature.MeterGroupByFilters["model"]
			modelTypeFilter, hasModelTypeFilter := feature.MeterGroupByFilters["type"]

			if hasProviderFilter && hasModelFilter && hasModelTypeFilter {
				// TODO filter models properly
				provider := providerFilter.Eq
				model := modelFilter.Eq
				costType := modelTypeFilter.Eq

				internalCostPerUnitFloat64, err := modelDev.getModelUnitCost(*provider, *model, CostType(*costType))
				if err == nil {
					internalCostPerUnit = alpacadecimal.NewFromFloat(internalCostPerUnitFloat64)
				}
			}

			// Calculate the cost for each window
			rows := make([]api.InvoiceLineCostRow, 0, len(usageRows))

			for _, row := range usageRows {
				usage := alpacadecimal.NewFromFloat(row.Value)
				cost := usage.Mul(costPerUnit)

				row := api.InvoiceLineCostRow{
					WindowStart: row.WindowStart,
					WindowEnd:   row.WindowEnd,
					Usage:       usage.String(),
					Cost:        cost.String(),
					CostPerUnit: costPerUnit.String(),
					GroupBy:     row.GroupBy,
				}

				if !internalCostPerUnit.IsZero() {
					internalCost := internalCostPerUnit.Mul(usage)
					row.InternalCostPerUnit = lo.ToPtr(internalCostPerUnit.String())
					row.InternalCost = lo.ToPtr(internalCost.String())
					totalInternalCost = totalInternalCost.Add(internalCost)
				}

				rows = append(rows, row)
			}

			response := api.InvoiceLineCost{
				From:        line.Period.Start,
				To:          line.Period.End,
				Currency:    string(line.Currency),
				CostPerUnit: costPerUnit.String(),
				Usage:       line.UsageBased.Quantity.String(),
				Cost:        line.Totals.Amount.String(),
				Rows:        rows,
			}

			if !totalInternalCost.IsZero() {
				response.InternalCost = lo.ToPtr(totalInternalCost.String())
				response.InternalCostPerUnit = lo.ToPtr(totalInternalCost.Div(*line.UsageBased.Quantity).String())
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetInvoiceLineCostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetInvoiceLineCost"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type ModelCostProvider struct {
	providers map[string]Provider
}

func (m *ModelCostProvider) initialize() error {
	url := "https://models.dev/api.json"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching JSON: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Use a generic structure since we don’t know the schema
	var data ModelsDevResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	m.providers = data

	return nil
}

// getModelUnitCost gets the unit cost of a model for a given cost type
func (m ModelCostProvider) getModelUnitCost(providerID string, modelID string, costType CostType) (float64, error) {
	provider, ok := m.providers[providerID]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("provider not found: %s", providerID))
	}

	model, ok := provider.Models[modelID]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("model not found: %s", modelID))
	}

	cost, ok := model.Cost[costType]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("cost type not found: %s", costType))
	}

	// Cost is per million tokens so we need to divide
	return cost / 1000000, nil
}

type ModelsDevResponse map[string]Provider

type Provider struct {
	Models map[string]Model `json:"models"`
}

type Model struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	Cost ModelCost `json:"cost"`
}

type ModelCost map[CostType]float64

type CostType string

const (
	CostTypeInputToken  CostType = "input"
	CostTypeOutputToken CostType = "output"
	CostTypeCacheRead   CostType = "cache_read"
	CostTypeCacheWrite  CostType = "cache_write"
)
