package featurecost

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	"github.com/openmeterio/openmeter/openmeter/cost"
)

// ConvertCostQueryResultToAPI converts a domain CostQueryResult to the v3 API response.
func ConvertCostQueryResultToAPI(result *cost.CostQueryResult, body api.MeterQueryRequest) api.FeatureCostQueryResult {
	if result == nil {
		return api.FeatureCostQueryResult{
			From: body.From,
			To:   body.To,
			Data: []api.FeatureCostQueryRow{},
		}
	}
	return api.FeatureCostQueryResult{
		From: body.From,
		To:   body.To,
		Data: lo.Map(result.Rows, func(row cost.CostQueryRow, _ int) api.FeatureCostQueryRow {
			return convertCostQueryRowToAPI(row)
		}),
	}
}

func convertCostQueryRowToAPI(row cost.CostQueryRow) api.FeatureCostQueryRow {
	dimensions := make(map[string]string)

	if row.Subject != nil {
		dimensions[query.DimensionSubject] = *row.Subject
	}

	if row.CustomerID != nil {
		dimensions[query.DimensionCustomerID] = *row.CustomerID
	}

	for key, value := range row.GroupBy {
		if key == query.DimensionSubject || key == query.DimensionCustomerID {
			continue
		}
		if value != nil {
			dimensions[key] = *value
		}
	}

	apiRow := api.FeatureCostQueryRow{
		Usage:      row.Usage.String(),
		Currency:   api.CurrencyCode(row.Currency),
		From:       row.WindowStart,
		To:         row.WindowEnd,
		Dimensions: dimensions,
	}

	if row.Cost != nil {
		apiRow.Cost = nullable.NewNullableWithValue(row.Cost.String())
	} else {
		apiRow.Cost = nullable.NewNullNullable[api.Numeric]()
	}

	if row.Detail != "" {
		apiRow.Detail = lo.ToPtr(row.Detail)
	}

	return apiRow
}
