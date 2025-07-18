package httpdriver

import (
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ToAPIMeter converts a meter.Meter to an api.Meter.
func ToAPIMeter(m meter.Meter) api.Meter {
	apiMeter := api.Meter{
		Id:            m.ID,
		Name:          &m.Name,
		Description:   m.Description,
		Slug:          m.Key,
		EventType:     m.EventType,
		EventFrom:     m.EventFrom,
		Aggregation:   api.MeterAggregation(m.Aggregation),
		ValueProperty: m.ValueProperty,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}

	if len(m.GroupBy) > 0 {
		apiMeter.GroupBy = &m.GroupBy
	}

	return apiMeter
}

// ToAPIMeterQueryResult constructs an api.MeterQueryResult
func ToAPIMeterQueryResult(from *time.Time, to *time.Time, windowSize *api.WindowSize, rows []meter.MeterQueryRow) api.MeterQueryResult {
	return api.MeterQueryResult{
		From:       from,
		To:         to,
		WindowSize: windowSize,
		Data:       ToAPIMeterQueryRowList(rows),
	}
}

// ToAPIMeterQueryRow converts a meter.MeterQueryRow to an api.MeterQueryRow.
func ToAPIMeterQueryRow(row meter.MeterQueryRow) api.MeterQueryRow {
	apiRow := api.MeterQueryRow{
		CustomerId:  row.CustomerID,
		Subject:     row.Subject,
		GroupBy:     row.GroupBy,
		WindowStart: row.WindowStart,
		WindowEnd:   row.WindowEnd,
		Value:       row.Value,
	}

	return apiRow
}

// ToAPIMeterQueryRowList converts a list of meter.MeterQueryRow to a list of api.MeterQueryRow.
func ToAPIMeterQueryRowList(rows []meter.MeterQueryRow) []api.MeterQueryRow {
	apiRows := make([]api.MeterQueryRow, len(rows))
	for i, row := range rows {
		apiRows[i] = ToAPIMeterQueryRow(row)
	}

	return apiRows
}

// ToQueryParamsFromAPIParams converts a api.QueryMeterParams to a streaming.QueryParams.
// This is used to convert an API POST query body to GET request params.
func ToRequestFromQueryParamsPOSTBody(apiParams api.QueryMeterParams) api.QueryMeterPostJSONRequestBody {
	// Map the POST request body to a GET request params
	request := api.QueryMeterPostJSONRequestBody{
		ClientId:         apiParams.ClientId,
		From:             apiParams.From,
		To:               apiParams.To,
		Subject:          apiParams.Subject,
		GroupBy:          apiParams.GroupBy,
		FilterCustomerId: apiParams.FilterCustomerId,
		WindowSize:       apiParams.WindowSize,
		WindowTimeZone:   apiParams.WindowTimeZone,
	}

	if apiParams.FilterGroupBy != nil {
		filterGroupBy := map[string][]string{}
		for k, v := range *apiParams.FilterGroupBy {
			filterGroupBy[k] = []string{v}
		}
		request.FilterGroupBy = &filterGroupBy
	}

	return request
}

// ToQueryParamsFromRequest converts a api.QueryMeterPostJSONRequestBody to a streaming.QueryParams.
// This is used to convert an API GET query params to a service level streaming.QueryParams.
func ToQueryParamsFromRequest(m meter.Meter, filterCustomer []customer.Customer, request api.QueryMeterPostJSONRequestBody) (streaming.QueryParams, error) {
	params := streaming.QueryParams{
		ClientID:       request.ClientId,
		From:           request.From,
		To:             request.To,
		FilterCustomer: filterCustomer,
	}

	if request.WindowSize != nil {
		params.WindowSize = lo.ToPtr(meter.WindowSize(*request.WindowSize))
	}

	if request.GroupBy != nil {
		for _, groupBy := range *request.GroupBy {
			// Validate group by, `subject` is a special group by
			if ok := groupBy == "subject" || groupBy == "customer_id" || m.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				return params, models.NewGenericValidationError(err)
			}

			params.GroupBy = append(params.GroupBy, groupBy)
		}
	}

	// Subject is a special query parameter which both filters and groups by subject(s)
	if request.Subject != nil {
		params.FilterSubject = *request.Subject

		// Add subject to group by if not already present
		if !slices.Contains(params.GroupBy, "subject") {
			params.GroupBy = append(params.GroupBy, "subject")
		}
	}

	// Add customer_id to group by if not already present and there are customers to filter by
	if len(filterCustomer) > 0 && !slices.Contains(params.GroupBy, "customer_id") {
		params.GroupBy = append(params.GroupBy, "customer_id")
	}

	if request.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*request.WindowTimeZone)
		if err != nil {
			err := fmt.Errorf("invalid time zone: %w", err)
			return params, models.NewGenericValidationError(err)
		}
		params.WindowTimeZone = tz
	}

	if request.FilterGroupBy != nil {
		for k, v := range *request.FilterGroupBy {
			// GroupBy filters
			if _, ok := m.GroupBy[k]; ok {
				if params.FilterGroupBy == nil {
					params.FilterGroupBy = map[string][]string{}
				}

				params.FilterGroupBy[k] = v
				continue
			} else {
				err := fmt.Errorf("invalid group by filter: %s", k)
				return params, models.NewGenericValidationError(err)
			}
		}
	}

	return params, nil
}
