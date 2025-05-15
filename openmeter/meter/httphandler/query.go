package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/samber/lo"
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

			params, err := ToQueryParamsFromAPIParams(meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			// We allow caching queries for HTTP requests
			// This alone is not enough for a query to be cached. The cache must be both enabled in the streaming connector,
			// and if any namespace template is provided to the cache, the namespace must match the template regex.
			params.Cachable = true

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
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
			subjects := []string{}
			for _, row := range rows {
				if row.Subject == nil {
					continue
				}

				subjects = append(subjects, *row.Subject)
			}

			// List customers for subjects to enrich results
			customersBySubjectKey := map[string]*customer.Customer{}

			if len(subjects) > 0 {
				customers, err := ListAllCustomers(ctx, h.customerService, customer.ListCustomersInput{
					Namespace: request.namespace,
					Subjects:  &subjects,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to list customers: %w", err)
				}

				for _, c := range customers {
					for _, key := range c.UsageAttribution.SubjectKeys {
						customersBySubjectKey[key] = &c
					}
				}
			}

			// Enrich rows with customer information
			enrichedRows := []enrichedMeterQueryRow{}

			for _, row := range rows {
				enrichedRow := enrichedMeterQueryRow{
					MeterQueryRow: row,
				}

				// If subject is not nil, try to enrich with customer information
				if row.Subject != nil {
					customer, ok := customersBySubjectKey[*row.Subject]
					if ok {
						enrichedRow.Customer = customer
					}
				}

				enrichedRows = append(enrichedRows, enrichedRow)
			}

			response := NewQueryMeterCSVResult(meter.Key, params.GroupBy, enrichedRows)

			return response, nil
		},
		commonhttp.CSVResponseEncoder[QueryMeterCSVResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterCSV"),
		)...,
	)
}

type enrichedMeterQueryRow struct {
	meter.MeterQueryRow
	Customer *customer.Customer
}

// QueryMeterCSVResult is a CSV response for query meter.
var _ QueryMeterCSVResponse = (*queryMeterCSVResult)(nil)

func NewQueryMeterCSVResult(meterSlug string, queryGroupBy []string, rows []enrichedMeterQueryRow) QueryMeterCSVResponse {
	return &queryMeterCSVResult{
		meterSlug:    meterSlug,
		queryGroupBy: queryGroupBy,
		rows:         rows,
	}
}

type queryMeterCSVResult struct {
	meterSlug    string
	queryGroupBy []string
	rows         []enrichedMeterQueryRow
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
	headers := []string{"window_start", "window_end", "subject", "customer_id", "customer_name", "customer_email"}
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

		if row.Customer != nil {
			data = append(data, row.Customer.ID, row.Customer.Name, lo.FromPtrOr(row.Customer.PrimaryEmail, ""))
		} else {
			data = append(data, "", "", "")
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

type QueryMeterPostRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterPostJSONRequestBody
}

type (
	QueryMeterPostParams   = string // meterIdOrSlug
	QueryMeterPostResponse = QueryMeterResponse
	QueryMeterPostHandler  httptransport.HandlerWithArgs[QueryMeterPostRequest, QueryMeterPostResponse, QueryMeterPostParams]
)

// QueryMeterPost returns a handler for query meter via POST.
func (h *handler) QueryMeterPost() QueryMeterPostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterIdOrSlug QueryMeterPostParams) (QueryMeterPostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterPostRequest{}, err
			}

			var request api.QueryMeterPostJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &request); err != nil {
				return QueryMeterPostRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			return QueryMeterPostRequest{
				namespace: ns,
				idOrSlug:  meterIdOrSlug,
				params:    request,
			}, nil
		},
		func(ctx context.Context, request QueryMeterPostRequest) (QueryMeterPostResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := ToQueryParamsFromRequest(meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			// We allow caching queries for HTTP requests
			params.Cachable = true

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterPostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterPost"),
		)...,
	)
}
