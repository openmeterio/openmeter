package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// QueryMeterCSVResult is a CSV response for query meter.
var _ QueryMeterCSVResponse = (*queryMeterCSVResult)(nil)

type (
	QueryMeterCSVParams   = QueryMeterParams
	QueryMeterCSVRequest  = QueryMeterRequest
	QueryMeterCSVResponse = commonhttp.CSVResponse
	QueryMeterCSVHandler  httptransport.HandlerWithArgs[QueryMeterCSVRequest, QueryMeterCSVResponse, QueryMeterCSVParams]
)

// QueryMeterCSV returns a handler for query meter.
func (h *handler) QueryMeterCSV() QueryMeterCSVHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryMeterCSVParams) (QueryMeterCSVRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterCSVRequest{}, err
			}

			return QueryMeterCSVRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
				params:    params.QueryMeterParams,
			}, nil
		},
		func(ctx context.Context, request QueryMeterCSVRequest) (QueryMeterCSVResponse, error) {
			// Get meter
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			// Query meter
			params, err := ToQueryParamsFromAPIParams(meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			// Collect subjects from query results if any
			subjectKeys := getSubjectsFromQueryResult(rows)

			var subjectsByKey map[string]subject.Subject

			// If there are subjects get the display names
			if len(subjectKeys) > 0 {
				subjects, err := h.subjectService.List(ctx, request.namespace, subject.ListParams{
					Keys: subjectKeys,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to get subjects: %w", err)
				}

				subjectsByKey = lo.KeyBy(subjects.Items, func(s subject.Subject) string {
					return s.Key
				})
			}

			response := NewQueryMeterCSVResult(meter.Key, params.GroupBy, rows, subjectsByKey)

			return response, nil
		},
		commonhttp.CSVResponseEncoder[QueryMeterCSVResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterCSV"),
		)...,
	)
}

// getSubjectsFromQueryResult returns the subjects from a query result.
func getSubjectsFromQueryResult(rows []meter.MeterQueryRow) []string {
	// Collect subjects from query results if any
	subjects := []string{}
	for _, row := range rows {
		if row.Subject == nil {
			continue
		}

		subjects = append(subjects, *row.Subject)
	}

	// Deduplicate subjects
	subjects = lo.Uniq(subjects)

	return subjects
}

func NewQueryMeterCSVResult(meterSlug string, queryGroupBy []string, rows []meter.MeterQueryRow, subjectsByKey map[string]subject.Subject) QueryMeterCSVResponse {
	return &queryMeterCSVResult{
		meterSlug:     meterSlug,
		queryGroupBy:  queryGroupBy,
		rows:          rows,
		subjectsByKey: subjectsByKey,
	}
}

type queryMeterCSVResult struct {
	meterSlug     string
	queryGroupBy  []string
	rows          []meter.MeterQueryRow
	subjectsByKey map[string]subject.Subject
}

// Records returns the CSV records.
func (a *queryMeterCSVResult) Records() [][]string {
	records := [][]string{}

	// Filter out the subject from the group by keys
	groupByKeys := []string{}
	for _, k := range a.queryGroupBy {
		if k == "subject" {
			continue
		}
		groupByKeys = append(groupByKeys, k)
	}

	// CSV headers
	headers := []string{"window_start", "window_end", "subject"}

	enhanceSubject := len(a.subjectsByKey) > 0

	if enhanceSubject {
		headers = append(headers, "subject_display_name")
	}

	if len(groupByKeys) > 0 {
		headers = append(headers, groupByKeys...)
	}
	headers = append(headers, "value")
	records = append(records, headers)

	// CSV data
	for _, row := range a.rows {
		data := []string{row.WindowStart.Format(time.RFC3339), row.WindowEnd.Format(time.RFC3339)}

		// Add subject if available
		if row.Subject != nil {
			data = append(data, *row.Subject)

			// Add display name if available
			if enhanceSubject {
				subject, ok := a.subjectsByKey[*row.Subject]
				if ok {
					data = append(data, lo.FromPtrOr(subject.DisplayName, ""))
				} else {
					data = append(data, "")
				}
			}
		} else {
			data = append(data, "")
		}

		for _, k := range groupByKeys {
			var groupByValue string

			if row.GroupBy[k] != nil {
				groupByValue = *row.GroupBy[k]
			}
			data = append(data, groupByValue)
		}
		data = append(data, fmt.Sprintf("%f", row.Value))
		records = append(records, data)
	}

	return records
}

// FileName returns the CSV file name.
func (a *queryMeterCSVResult) FileName() string {
	return fmt.Sprintf("%s.csv", a.meterSlug)
}
