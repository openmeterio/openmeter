package featurecost

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	QueryFeatureCostCSVRequest  = QueryFeatureCostRequest
	QueryFeatureCostCSVResponse = commonhttp.CSVResponse
	QueryFeatureCostCSVParams   = QueryFeatureCostParams
	QueryFeatureCostCSVHandler  = httptransport.HandlerWithArgs[QueryFeatureCostCSVRequest, QueryFeatureCostCSVResponse, QueryFeatureCostCSVParams]
)

func (h *handler) QueryFeatureCostCSV() QueryFeatureCostCSVHandler {
	return httptransport.NewHandlerWithArgs(
		h.decodeQueryFeatureCostRequest,
		func(ctx context.Context, req QueryFeatureCostCSVRequest) (QueryFeatureCostCSVResponse, error) {
			result, err := h.queryFeatureCost(ctx, req)
			if err != nil {
				return nil, err
			}

			var rows []cost.CostQueryRow
			if result.result != nil {
				rows = result.result.Rows
			}

			var customerIDs []string
			if slices.Contains(result.groupBy, query.DimensionCustomerID) {
				for _, row := range rows {
					if row.CustomerID != nil {
						customerIDs = append(customerIDs, *row.CustomerID)
					}
				}
			}

			var customersByID map[string]customer.Customer
			if len(customerIDs) > 0 {
				customers, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
					Namespace:   req.Namespace,
					CustomerIDs: lo.Uniq(customerIDs),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to list customers for csv enrichment: %w", err)
				}
				customersByID = lo.KeyBy(customers.Items, func(c customer.Customer) string { return c.ID })
			}

			return &queryFeatureCostCSVResult{
				featureKey:    result.featureKey,
				groupBy:       result.groupBy,
				rows:          rows,
				customersByID: customersByID,
			}, nil
		},
		commonhttp.CSVResponseEncoder[QueryFeatureCostCSVResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-feature-cost-csv"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

type queryFeatureCostCSVResult struct {
	featureKey    string
	groupBy       []string
	rows          []cost.CostQueryRow
	customersByID map[string]customer.Customer
}

var _ commonhttp.CSVResponse = &queryFeatureCostCSVResult{}

func (r *queryFeatureCostCSVResult) FileName() string {
	return r.featureKey + "-cost"
}

func (r *queryFeatureCostCSVResult) Records() [][]string {
	hasSubject := slices.Contains(r.groupBy, query.DimensionSubject)
	hasCustomer := slices.Contains(r.groupBy, query.DimensionCustomerID)

	headers := []string{"from", "to"}
	if hasSubject {
		headers = append(headers, "subject")
	}
	if hasCustomer {
		headers = append(headers, "customer_id", "customer_key", "customer_name")
	}

	var dimensions []string
	for _, key := range r.groupBy {
		if key != query.DimensionSubject && key != query.DimensionCustomerID {
			dimensions = append(dimensions, key)
		}
	}
	headers = append(headers, dimensions...)
	headers = append(headers, "usage", "cost", "currency", "detail")

	records := make([][]string, 0, len(r.rows)+1)
	records = append(records, headers)
	for _, row := range r.rows {
		record := make([]string, 0, len(headers))
		record = append(record, row.WindowStart.Format(time.RFC3339), row.WindowEnd.Format(time.RFC3339))
		if hasSubject {
			record = append(record, lo.FromPtr(row.Subject))
		}
		if hasCustomer {
			id := lo.FromPtr(row.CustomerID)
			c := r.customersByID[id]
			record = append(record, id, lo.FromPtr(c.Key), c.Name)
		}
		for _, key := range dimensions {
			record = append(record, lo.FromPtr(row.GroupBy[key]))
		}

		var costAmount string
		if row.Cost != nil {
			costAmount = row.Cost.String()
		}
		record = append(record, row.Usage.String(), costAmount, string(row.Currency), row.Detail)
		records = append(records, record)
	}
	return records
}
