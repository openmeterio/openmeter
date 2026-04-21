package meters

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	QueryMeterCSVRequest  = QueryMeterRequest
	QueryMeterCSVResponse = commonhttp.CSVResponse
	QueryMeterCSVParams   = QueryMeterParams
	QueryMeterCSVHandler  httptransport.HandlerWithArgs[QueryMeterCSVRequest, QueryMeterCSVResponse, QueryMeterCSVParams]
)

const (
	csvColumnFrom         = "from"
	csvColumnTo           = "to"
	csvColumnValue        = "value"
	csvColumnCustomerID   = "customer_id"
	csvColumnCustomerKey  = "customer_key"
	csvColumnCustomerName = "customer_name"
)

func (h *handler) QueryMeterCSV() QueryMeterCSVHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterID QueryMeterCSVParams) (QueryMeterCSVRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterCSVRequest{}, err
			}

			var body api.MeterQueryRequest
			if err := request.ParseBody(r, &body); err != nil {
				return QueryMeterCSVRequest{}, err
			}

			return QueryMeterCSVRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        meterID,
				},
				Body: body,
			}, nil
		},
		func(ctx context.Context, req QueryMeterCSVRequest) (QueryMeterCSVResponse, error) {
			m, err := h.service.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: req.Namespace,
				IDOrSlug:  req.ID,
			})
			if err != nil {
				return nil, err
			}

			params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))
			if err != nil {
				return nil, err
			}

			rows, err := h.streaming.QueryMeter(ctx, req.Namespace, m, params)
			if err != nil {
				return nil, err
			}

			// Enrich with customer metadata (key, name) when any row has a CustomerID.
			customerIDs := collectCustomerIDs(rows)

			var customersByID map[string]customer.Customer
			if len(customerIDs) > 0 {
				result, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
					Namespace:   req.Namespace,
					CustomerIDs: customerIDs,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to list customers for csv enrichment: %w", err)
				}

				customersByID = lo.KeyBy(result.Items, func(c customer.Customer) string {
					return c.ID
				})
			}

			return newQueryMeterCSVResult(m.Key, params.GroupBy, rows, customersByID), nil
		},
		commonhttp.CSVResponseEncoder[QueryMeterCSVResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-meter-csv"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// collectCustomerIDs returns the distinct non-nil customer IDs referenced by the rows.
func collectCustomerIDs(rows []meter.MeterQueryRow) []string {
	ids := make([]string, 0)
	for _, row := range rows {
		if row.CustomerID == nil {
			continue
		}
		ids = append(ids, *row.CustomerID)
	}
	return lo.Uniq(ids)
}

// Column order:
//
//	from, to,
//	[subject,]
//	[customer_id, customer_key, customer_name,]
//	<other dimensions...>,
//	value
type queryMeterCSVResult struct {
	meterSlug     string
	groupBy       []string
	rows          []meter.MeterQueryRow
	customersByID map[string]customer.Customer
}

var _ commonhttp.CSVResponse = &queryMeterCSVResult{}

func newQueryMeterCSVResult(
	meterSlug string,
	groupBy []string,
	rows []meter.MeterQueryRow,
	customersByID map[string]customer.Customer,
) *queryMeterCSVResult {
	return &queryMeterCSVResult{
		meterSlug:     meterSlug,
		groupBy:       groupBy,
		rows:          rows,
		customersByID: customersByID,
	}
}

func (r *queryMeterCSVResult) FileName() string {
	return r.meterSlug
}

func (r *queryMeterCSVResult) Records() [][]string {
	hasSubjectColumn := slices.Contains(r.groupBy, query.DimensionSubject)
	hasCustomerColumns := slices.Contains(r.groupBy, query.DimensionCustomerID)

	otherDimensions := make([]string, 0, len(r.groupBy))
	for _, k := range r.groupBy {
		switch k {
		case query.DimensionSubject, query.DimensionCustomerID:
			// Handled as reserved columns.
		default:
			otherDimensions = append(otherDimensions, k)
		}
	}

	headers := []string{csvColumnFrom, csvColumnTo}
	if hasSubjectColumn {
		headers = append(headers, query.DimensionSubject)
	}
	if hasCustomerColumns {
		headers = append(headers, csvColumnCustomerID, csvColumnCustomerKey, csvColumnCustomerName)
	}
	headers = append(headers, otherDimensions...)
	headers = append(headers, csvColumnValue)

	records := make([][]string, 0, len(r.rows)+1)
	records = append(records, headers)

	for _, row := range r.rows {
		record := make([]string, 0, len(headers))
		record = append(record,
			row.WindowStart.Format(time.RFC3339),
			row.WindowEnd.Format(time.RFC3339),
		)

		if hasSubjectColumn {
			record = append(record, lo.FromPtrOr(row.Subject, ""))
		}

		if hasCustomerColumns {
			var id, key, name string
			if row.CustomerID != nil {
				id = *row.CustomerID
				if c, ok := r.customersByID[id]; ok {
					key = lo.FromPtrOr(c.Key, "")
					name = c.Name
				}
			}
			record = append(record,
				id,
				key,
				name,
			)
		}

		for _, k := range otherDimensions {
			var v string
			if ptr, ok := row.GroupBy[k]; ok && ptr != nil {
				v = *ptr
			}
			record = append(record, v)
		}

		record = append(record, strconv.FormatFloat(row.Value, 'f', -1, 64))
		records = append(records, record)
	}

	return records
}
