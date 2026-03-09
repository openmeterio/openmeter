package meters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

const maxGroupByFilterComplexityDepth = 2

type (
	QueryMeterRequest struct {
		models.NamespacedID
		Body api.MeterQueryRequest
	}
	QueryMeterResponse = api.MeterQueryResult
	QueryMeterParams   = string
	QueryMeterHandler  httptransport.HandlerWithArgs[QueryMeterRequest, QueryMeterResponse, QueryMeterParams]
)

func (h *handler) QueryMeter() QueryMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID QueryMeterParams) (QueryMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterRequest{}, err
			}

			var body api.MeterQueryRequest
			if err := request.ParseBody(r, &body); err != nil {
				return QueryMeterRequest{}, err
			}

			return QueryMeterRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        meterID,
				},
				Body: body,
			}, nil
		},
		func(ctx context.Context, req QueryMeterRequest) (QueryMeterResponse, error) {
			m, err := h.service.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: req.Namespace,
				IDOrSlug:  req.ID,
			})
			if err != nil {
				return QueryMeterResponse{}, err
			}

			params, err := h.buildQueryParams(ctx, m, req.Body)
			if err != nil {
				return QueryMeterResponse{}, err
			}

			rows, err := h.streaming.QueryMeter(ctx, req.Namespace, m, params)
			if err != nil {
				return QueryMeterResponse{}, err
			}

			return ConvertMeterQueryResultToAPI(req.Body.From, req.Body.To, rows), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-meter"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func (h *handler) buildQueryParams(ctx context.Context, m meter.Meter, body api.MeterQueryRequest) (streaming.QueryParams, error) {
	params := streaming.QueryParams{
		From: body.From,
		To:   body.To,
	}

	if body.Granularity != nil {
		ws, err := ConvertISO8601DurationToWindowSize(string(*body.Granularity))
		if err != nil {
			return params, err
		}
		params.WindowSize = &ws
	}

	if body.TimeZone != nil {
		tz, err := time.LoadLocation(*body.TimeZone)
		if err != nil {
			return params, NewInvalidTimeZoneError(*body.TimeZone)
		}
		params.WindowTimeZone = tz
	}

	if body.GroupByDimensions != nil {
		for _, groupBy := range *body.GroupByDimensions {
			if !isSupportedGroupByDimension(m, groupBy) {
				return params, NewInvalidGroupByError(groupBy)
			}
			params.GroupBy = append(params.GroupBy, groupBy)
		}
	}

	if filters := body.Filters; filters != nil {
		if dimensions := filters.Dimensions; dimensions != nil {
			if dimensions.Subject != nil {
				// TODO: migrate to filter.FilterString
				subjects, err := ExtractStringsFromQueryFilter(dimensions.Subject, "dimensions", dimensionSubject)
				if err != nil {
					return params, err
				}

				params.FilterSubject = subjects

				if len(subjects) > 0 && !slices.Contains(params.GroupBy, dimensionSubject) {
					params.GroupBy = append(params.GroupBy, dimensionSubject)
				}
			}

			if dimensions.CustomerId != nil {
				// TODO: migrate to filter.FilterString
				customerIDs, err := ExtractStringsFromQueryFilter(dimensions.CustomerId, "dimensions", dimensionCustomerID)
				if err != nil {
					return params, err
				}

				if len(customerIDs) > 0 {
					filterCustomers, err := h.resolveCustomers(ctx, m.Namespace, customerIDs)
					if err != nil {
						return params, err
					}

					params.FilterCustomer = lo.Map(filterCustomers, func(c customer.Customer, _ int) streaming.Customer {
						return c
					})

					if !slices.Contains(params.GroupBy, dimensionCustomerID) {
						params.GroupBy = append(params.GroupBy, dimensionCustomerID)
					}
				}
			}

			if len(dimensions.AdditionalProperties) > 0 {
				params.FilterGroupBy = make(map[string]filter.FilterString, len(dimensions.AdditionalProperties))
				for k, v := range dimensions.AdditionalProperties {
					if _, ok := m.GroupBy[k]; !ok {
						return params, NewInvalidDimensionFilterError(k)
					}
					f := request.ConvertQueryFilterStringMapItem(v)
					if err := f.ValidateWithComplexity(maxGroupByFilterComplexityDepth); err != nil {
						return params, models.NewGenericValidationError(fmt.Errorf("dimension filter %q: %w", k, err))
					}
					params.FilterGroupBy[k] = f
				}
			}
		}
	}

	return params, nil
}

func (h *handler) resolveCustomers(ctx context.Context, namespace string, customerIDs []string) ([]customer.Customer, error) {
	if len(customerIDs) == 0 {
		return nil, nil
	}

	customers, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:      namespace,
		CustomerIDs:    customerIDs,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	customersById := lo.KeyBy(customers.Items, func(c customer.Customer) string {
		return c.ID
	})

	var errs []error
	for _, id := range customerIDs {
		if _, ok := customersById[id]; !ok {
			errs = append(errs, NewCustomerNotFoundError(id))
		}
	}

	return customers.Items, errors.Join(errs...)
}
