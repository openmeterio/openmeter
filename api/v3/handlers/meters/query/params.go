package query

import (
	"context"
	"fmt"
	"slices"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

const maxGroupByFilterComplexityDepth = 2

// BuildQueryParams converts a v3 MeterQueryRequest body into streaming.QueryParams.
// It validates groupBy dimensions against the given meter and resolves customer filters.
func BuildQueryParams(ctx context.Context, m meter.Meter, body api.MeterQueryRequest, resolveCustomers CustomerResolverFunc) (streaming.QueryParams, error) {
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
			if !IsSupportedGroupByDimension(m, groupBy) {
				return params, NewInvalidGroupByError(groupBy)
			}
			if !slices.Contains(params.GroupBy, groupBy) {
				params.GroupBy = append(params.GroupBy, groupBy)
			}
		}
	}

	if filters := body.Filters; filters != nil {
		if dimensions := filters.Dimensions; dimensions != nil {
			for k, v := range *dimensions {
				switch k {
				case DimensionSubject:
					if err := ValidateQueryFilterStringMapItem(v, "dimensions", DimensionSubject); err != nil {
						return params, err
					}

					subjects, err := ExtractStringsFromQueryFilterMapItem(&v, "dimensions", DimensionSubject)
					if err != nil {
						return params, err
					}

					params.FilterSubject = subjects

					if len(subjects) > 0 && !slices.Contains(params.GroupBy, DimensionSubject) {
						params.GroupBy = append(params.GroupBy, DimensionSubject)
					}

				case DimensionCustomerID:
					if err := ValidateQueryFilterStringMapItem(v, "dimensions", DimensionCustomerID); err != nil {
						return params, err
					}

					customerIDs, err := ExtractStringsFromQueryFilterMapItem(&v, "dimensions", DimensionCustomerID)
					if err != nil {
						return params, err
					}

					if len(customerIDs) > 0 {
						filterCustomers, err := resolveCustomers(ctx, m.Namespace, customerIDs)
						if err != nil {
							return params, err
						}

						params.FilterCustomer = CustomersToStreaming(filterCustomers)

						if !slices.Contains(params.GroupBy, DimensionCustomerID) {
							params.GroupBy = append(params.GroupBy, DimensionCustomerID)
						}
					}

				default:
					if _, ok := m.GroupBy[k]; !ok {
						return params, NewInvalidDimensionFilterError(k)
					}
					if err := ValidateQueryFilterStringMapItem(v, "dimensions", k); err != nil {
						return params, err
					}
					f := request.ConvertQueryFilterStringMapItem(v)
					if err := f.ValidateWithComplexity(maxGroupByFilterComplexityDepth); err != nil {
						return params, models.NewGenericValidationError(fmt.Errorf("dimension filter %q: %w", k, err))
					}
					if params.FilterGroupBy == nil {
						params.FilterGroupBy = make(map[string]filter.FilterString)
					}
					params.FilterGroupBy[k] = f
				}
			}
		}
	}

	return params, nil
}
