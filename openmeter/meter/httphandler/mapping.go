package httpdriver

import (
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
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
func ToAPIMeterQueryResult(apiParams api.QueryMeterParams, rows []meter.MeterQueryRow) api.MeterQueryResult {
	return api.MeterQueryResult{
		From:       apiParams.From,
		To:         apiParams.To,
		WindowSize: apiParams.WindowSize,
		Data:       ToAPIMeterQueryRowList(rows),
	}
}

// ToAPIMeterQueryRow converts a meter.MeterQueryRow to an api.MeterQueryRow.
func ToAPIMeterQueryRow(row meter.MeterQueryRow) api.MeterQueryRow {
	apiRow := api.MeterQueryRow{
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

// ToQueryMeterParams converts a api.QueryMeterParams to a streaming.QueryParams.
func ToQueryMeterParams(m meter.Meter, apiParams api.QueryMeterParams) (streaming.QueryParams, error) {
	params := streaming.QueryParams{
		ClientID: apiParams.ClientId,
		From:     apiParams.From,
		To:       apiParams.To,
	}

	if apiParams.WindowSize != nil {
		params.WindowSize = lo.ToPtr(meter.WindowSize(*apiParams.WindowSize))
	}

	if apiParams.GroupBy != nil {
		for _, groupBy := range *apiParams.GroupBy {
			// Validate group by, `subject` is a special group by
			if ok := groupBy == "subject" || m.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				return params, models.NewGenericValidationError(err)
			}

			params.GroupBy = append(params.GroupBy, groupBy)
		}
	}

	// Subject is a special query parameter which both filters and groups by subject(s)
	if apiParams.Subject != nil {
		params.FilterSubject = *apiParams.Subject

		// Add subject to group by if not already present
		if !slices.Contains(params.GroupBy, "subject") {
			params.GroupBy = append(params.GroupBy, "subject")
		}
	}

	if apiParams.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*apiParams.WindowTimeZone)
		if err != nil {
			err := fmt.Errorf("invalid time zone: %w", err)
			return params, models.NewGenericValidationError(err)
		}
		params.WindowTimeZone = tz
	}

	if apiParams.FilterGroupBy != nil {
		for k, v := range *apiParams.FilterGroupBy {
			// GroupBy filters
			if _, ok := m.GroupBy[k]; ok {
				if params.FilterGroupBy == nil {
					params.FilterGroupBy = map[string][]string{}
				}

				params.FilterGroupBy[k] = []string{v}
				continue
			} else {
				err := fmt.Errorf("invalid group by filter: %s", k)
				return params, models.NewGenericValidationError(err)
			}
		}
	}

	return params, nil
}
