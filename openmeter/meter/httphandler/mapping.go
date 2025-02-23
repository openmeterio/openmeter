package httpdriver

import (
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// ToAPIMeter converts a meter.Meter to an api.Meter.
func ToAPIMeter(m meter.Meter) api.Meter {
	apiMeter := api.Meter{
		Id:          m.ID,
		Slug:        m.Slug,
		EventType:   m.EventType,
		WindowSize:  lo.ToPtr(api.WindowSize(m.WindowSize)),
		Aggregation: api.MeterAggregation(m.Aggregation),
	}

	if m.Description != "" {
		apiMeter.Description = &m.Description
	}

	if m.ValueProperty != "" {
		apiMeter.ValueProperty = &m.ValueProperty
	}

	if len(m.GroupBy) > 0 {
		apiMeter.GroupBy = &m.GroupBy
	}

	return apiMeter
}

// toAPIMeterQueryResult constructs an api.MeterQueryResult
func toAPIMeterQueryResult(apiParams api.QueryMeterParams, rows []meterpkg.MeterQueryRow) api.MeterQueryResult {
	return api.MeterQueryResult{
		From:       apiParams.From,
		To:         apiParams.To,
		WindowSize: apiParams.WindowSize,
		Data:       toAPIMeterQueryRowList(rows),
	}
}

// toAPIMeterQueryRow converts a meterpkg.MeterQueryRow to an api.MeterQueryRow.
func toAPIMeterQueryRow(row meterpkg.MeterQueryRow) api.MeterQueryRow {
	apiRow := api.MeterQueryRow{
		Subject:     row.Subject,
		GroupBy:     row.GroupBy,
		WindowStart: row.WindowStart,
		WindowEnd:   row.WindowEnd,
		Value:       row.Value,
	}

	return apiRow
}

// toAPIMeterQueryRowList converts a list of meterpkg.MeterQueryRow to a list of api.MeterQueryRow.
func toAPIMeterQueryRowList(rows []meterpkg.MeterQueryRow) []api.MeterQueryRow {
	apiRows := make([]api.MeterQueryRow, len(rows))
	for i, row := range rows {
		apiRows[i] = toAPIMeterQueryRow(row)
	}

	return apiRows
}

// toQueryMeterParams converts a api.QueryMeterParams to a streaming.QueryParams.
func toQueryMeterParams(meter meter.Meter, apiParams api.QueryMeterParams) (streaming.QueryParams, error) {
	params := streaming.QueryParams{
		From: apiParams.From,
		To:   apiParams.To,
	}

	if apiParams.WindowSize != nil {
		params.WindowSize = lo.ToPtr(meterpkg.WindowSize(*apiParams.WindowSize))
	}

	if apiParams.GroupBy != nil {
		for _, groupBy := range *apiParams.GroupBy {
			// Validate group by, `subject` is a special group by
			if ok := groupBy == "subject" || meter.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				return params, meterpkg.NewBadRequestError(err)
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
			return params, meterpkg.NewBadRequestError(err)
		}
		params.WindowTimeZone = tz
	}

	if apiParams.FilterGroupBy != nil {
		for k, v := range *apiParams.FilterGroupBy {
			// GroupBy filters
			if _, ok := meter.GroupBy[k]; ok {
				if params.FilterGroupBy == nil {
					params.FilterGroupBy = map[string][]string{}
				}

				params.FilterGroupBy[k] = []string{v}
				continue
			} else {
				err := fmt.Errorf("invalid group by filter: %s", k)
				return params, meterpkg.NewBadRequestError(err)
			}
		}
	}

	return params, nil
}
