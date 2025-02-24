package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListSubjectsResponse = *[]string
	ListSubjectsHandler  httptransport.HandlerWithArgs[ListSubjectsRequest, ListSubjectsResponse, ListSubjectsParams]
)

type ListSubjectsParams struct {
	IdOrSlug string
}

type ListSubjectsRequest struct {
	namespace string
	idOrSlug  string
}

// ListSubjects returns a handler for query meter.
func (h *handler) ListSubjects() ListSubjectsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubjectsParams) (ListSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubjectsRequest{}, err
			}

			return ListSubjectsRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
			}, nil
		},
		func(ctx context.Context, request ListSubjectsRequest) (ListSubjectsResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			subjectKeys, err := h.streaming.ListMeterSubjects(ctx, request.namespace, meter, streaming.ListMeterSubjectsParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to list subjects: %w", err)
			}

			return &subjectKeys, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubjectsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	QueryMeterResponse = *api.MeterQueryResult
	QueryMeterHandler  httptransport.HandlerWithArgs[QueryMeterRequest, QueryMeterResponse, QueryMeterParams]
)

type QueryMeterParams struct {
	api.QueryMeterParams
	IdOrSlug string
}

type QueryMeterRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterParams
}

// QueryMeter returns a handler for query meter.
func (h *handler) QueryMeter() QueryMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryMeterParams) (QueryMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterRequest{}, err
			}

			return QueryMeterRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
				params:    params.QueryMeterParams,
			}, nil
		},
		func(ctx context.Context, request QueryMeterRequest) (QueryMeterResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := toQueryMeterParams(meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := toAPIMeterQueryResult(request.params, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

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
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := toQueryMeterParams(meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := NewQueryMeterCSVResult(meter.Slug, params.GroupBy, rows)

			return response, nil
		},
		commonhttp.CSVResponseEncoder[QueryMeterCSVResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterCSV"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// QueryMeterCSVResult is a CSV response for query meter.
var _ QueryMeterCSVResponse = (*queryMeterCSVResult)(nil)

func NewQueryMeterCSVResult(meterSlug string, queryGroupBy []string, rows []meter.MeterQueryRow) QueryMeterCSVResponse {
	return &queryMeterCSVResult{
		meterSlug:    meterSlug,
		queryGroupBy: queryGroupBy,
		rows:         rows,
	}
}

type queryMeterCSVResult struct {
	meterSlug    string
	queryGroupBy []string
	rows         []meter.MeterQueryRow
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
	if len(groupByKeys) > 0 {
		headers = append(headers, groupByKeys...)
	}
	headers = append(headers, "value")
	records = append(records, headers)

	// CSV data
	for _, row := range a.rows {
		data := []string{row.WindowStart.Format(time.RFC3339), row.WindowEnd.Format(time.RFC3339)}
		if row.Subject != nil {
			data = append(data, *row.Subject)
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
