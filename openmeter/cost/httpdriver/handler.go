package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CostHandler interface {
	QueryFeatureCost() QueryFeatureCostHandler
}

type costHandlers struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	costService      cost.Service
	customerService  customer.Service
}

func NewCostHandler(
	costService cost.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) CostHandler {
	return &costHandlers{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		costService:      costService,
		customerService:  customerService,
	}
}

type QueryFeatureCostHandlerRequest struct {
	Namespace string
	FeatureID string
	Params    api.QueryFeatureCostParams
}

type (
	QueryFeatureCostHandlerResponse = *api.FeatureCostQueryResult
	QueryFeatureCostHandlerParams   struct {
		FeatureID string
		api.QueryFeatureCostParams
	}
)

type QueryFeatureCostHandler httptransport.HandlerWithArgs[QueryFeatureCostHandlerRequest, QueryFeatureCostHandlerResponse, QueryFeatureCostHandlerParams]

func (h *costHandlers) QueryFeatureCost() QueryFeatureCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, handlerParams QueryFeatureCostHandlerParams) (QueryFeatureCostHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryFeatureCostHandlerRequest{}, err
			}

			return QueryFeatureCostHandlerRequest{
				Namespace: ns,
				FeatureID: handlerParams.FeatureID,
				Params:    handlerParams.QueryFeatureCostParams,
			}, nil
		},
		func(ctx context.Context, req QueryFeatureCostHandlerRequest) (QueryFeatureCostHandlerResponse, error) {
			// Build streaming query params from HTTP request
			params, err := h.buildCostQueryParams(ctx, req)
			if err != nil {
				return nil, err
			}

			// Delegate to service
			result, err := h.costService.QueryFeatureCost(ctx, cost.QueryFeatureCostInput{
				Namespace:   req.Namespace,
				FeatureID:   req.FeatureID,
				QueryParams: params,
			})
			if err != nil {
				// TODO: remove after we return generic not found error
				// instead of feature.FeatureNotFoundError
				if _, ok := lo.ErrorsAs[*feature.FeatureNotFoundError](err); ok {
					return nil, models.NewGenericNotFoundError(err)
				}

				return nil, err
			}

			// Map domain result to API response
			return mapCostQueryResultToAPI(result, req.Params), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryFeatureCostHandlerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryFeatureCost"),
			httptransport.WithErrorEncoder(commonhttp.GenericErrorEncoder()),
		)...,
	)
}

// mapCostQueryResultToAPI maps a domain CostQueryResult to the API response type.
func mapCostQueryResultToAPI(result *cost.CostQueryResult, params api.QueryFeatureCostParams) *api.FeatureCostQueryResult {
	apiRows := make([]api.FeatureCostQueryRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		apiRow := api.FeatureCostQueryRow{
			Usage:       row.Usage.String(),
			Currency:    row.Currency,
			WindowStart: row.WindowStart,
			WindowEnd:   row.WindowEnd,
			Subject:     row.Subject,
			CustomerId:  row.CustomerID,
		}

		if row.Cost != nil {
			costStr := row.Cost.String()
			apiRow.Cost = &costStr
		}

		if row.Detail != "" {
			apiRow.Detail = lo.ToPtr(row.Detail)
		}

		if len(row.GroupBy) > 0 {
			apiRow.GroupBy = &row.GroupBy
		}

		apiRows = append(apiRows, apiRow)
	}

	return &api.FeatureCostQueryResult{
		From:       params.From,
		To:         params.To,
		WindowSize: params.WindowSize,
		Currency:   result.Currency,
		Data:       apiRows,
	}
}

// buildCostQueryParams converts API query params to streaming.QueryParams.
func (h *costHandlers) buildCostQueryParams(ctx context.Context, req QueryFeatureCostHandlerRequest) (streaming.QueryParams, error) {
	qp := req.Params

	params := streaming.QueryParams{
		ClientID: qp.ClientId,
		From:     qp.From,
		To:       qp.To,
	}

	if qp.WindowSize != nil {
		params.WindowSize = lo.ToPtr(meter.WindowSize(*qp.WindowSize))
	}

	if qp.GroupBy != nil {
		params.GroupBy = append(params.GroupBy, *qp.GroupBy...)
	}

	// Subject is a special query parameter which both filters and groups by subject(s)
	if qp.Subject != nil {
		params.FilterSubject = *qp.Subject
		if !slices.Contains(params.GroupBy, "subject") {
			params.GroupBy = append(params.GroupBy, "subject")
		}
	}

	// Resolve filter customer IDs
	if qp.FilterCustomerId != nil {
		filterCustomer, err := h.resolveFilterCustomers(ctx, req.Namespace, *qp.FilterCustomerId)
		if err != nil {
			return params, err
		}
		params.FilterCustomer = filterCustomer
		if len(filterCustomer) > 0 && !slices.Contains(params.GroupBy, "customer_id") {
			params.GroupBy = append(params.GroupBy, "customer_id")
		}
	}

	if qp.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*qp.WindowTimeZone)
		if err != nil {
			return params, models.NewGenericValidationError(fmt.Errorf("invalid time zone: %w", err))
		}
		params.WindowTimeZone = tz
	}

	if qp.AdvancedMeterGroupByFilters != nil && len(*qp.AdvancedMeterGroupByFilters) > 0 {
		params.FilterGroupBy = apiconverter.ConvertStringMap(*qp.AdvancedMeterGroupByFilters)
	}

	if qp.FilterGroupBy != nil {
		filterGroupBy := map[string]string(*qp.FilterGroupBy)
		if len(filterGroupBy) > 0 {
			if len(params.FilterGroupBy) > 0 {
				return params, models.NewGenericValidationError(errors.New("advanced meter group by filters and filter group by cannot be used together"))
			}

			params.FilterGroupBy = make(map[string]filter.FilterString, len(filterGroupBy))
			for k, v := range filterGroupBy {
				params.FilterGroupBy[k] = filter.FilterString{
					Eq: lo.ToPtr(v),
				}
			}
		}
	}

	return params, nil
}

// resolveFilterCustomers resolves customer IDs to streaming.Customer.
func (h *costHandlers) resolveFilterCustomers(ctx context.Context, namespace string, customerIDs []string) ([]streaming.Customer, error) {
	if len(customerIDs) == 0 {
		return nil, nil
	}

	customers, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:   namespace,
		CustomerIDs: customerIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	customersById := lo.KeyBy(customers.Items, func(c customer.Customer) string {
		return c.ID
	})

	for _, id := range customerIDs {
		if _, ok := customersById[id]; !ok {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("customer with id %s not found", id))
		}
	}

	return lo.Map(customers.Items, func(c customer.Customer, _ int) streaming.Customer {
		return c
	}), nil
}

func (h *costHandlers) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
