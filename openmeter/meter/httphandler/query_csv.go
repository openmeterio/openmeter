package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	QueryMeterCSVParams   = QueryMeterParams
	QueryMeterCSVRequest  = QueryMeterRequest
	QueryMeterCSVResponse = commonhttp.CSVResponse
	QueryMeterCSVHandler  httptransport.HandlerWithArgs[QueryMeterCSVRequest, QueryMeterCSVResponse, QueryMeterCSVParams]
)

type enrichedMeterQueryRow struct {
	meter.MeterQueryRow
	Customer *customer.Customer
}

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
			subjects := getSubjectsFromQueryResult(rows)

			// List customers for subjects to enrich results
			customersBySubjectKey, err := listCustomersBySubjectKey(
				ctx,
				h.customerService,
				request.namespace,
				subjects,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get customers by subject key: %w", err)
			}

			// Enrich rows with customer information
			enrichedRows := enrichRowsWithCustomerInfo(rows, customersBySubjectKey)

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

// QueryMeterCSVResult is a CSV response for query meter.
var _ QueryMeterCSVResponse = (*queryMeterCSVResult)(nil)

// NewQueryMeterCSVResult creates a new query meter CSV result.
func NewQueryMeterCSVResult(meterSlug string, queryGroupBy []string, rows []enrichedMeterQueryRow) QueryMeterCSVResponse {
	return &queryMeterCSVResult{
		meterSlug:    meterSlug,
		queryGroupBy: queryGroupBy,
		rows:         rows,
	}
}

// queryMeterCSVResult is a CSV response for query meter.
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
	headers := []string{"window_start", "window_end", "subject", "customer_name", "customer_email"}
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
			data = append(data, row.Customer.Name, lo.FromPtrOr(row.Customer.PrimaryEmail, ""))
		} else {
			data = append(data, "", "")
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

// enrichRowsWithCustomerInfo enriches a list of rows with customer information.
func enrichRowsWithCustomerInfo(
	rows []meter.MeterQueryRow,
	customersBySubjectKey map[string]*customer.Customer,
) []enrichedMeterQueryRow {
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

	return enrichedRows
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
