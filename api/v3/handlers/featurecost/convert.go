package featurecost

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/query"
	"github.com/openmeterio/openmeter/openmeter/cost"
)

// ConvertCostQueryResultToAPI converts a domain CostQueryResult to the v3 API response.
func ConvertCostQueryResultToAPI(result *cost.CostQueryResult, body api.MeterQueryRequest) api.FeatureCostQueryResult {
	return api.FeatureCostQueryResult{
		From:     body.From,
		To:       body.To,
		Currency: api.CurrencyCode(result.Currency),
		Data: lo.Map(result.Rows, func(row cost.CostQueryRow, _ int) api.FeatureCostQueryRow {
			return convertCostQueryRowToAPI(row)
		}),
	}
}

func convertCostQueryRowToAPI(row cost.CostQueryRow) api.FeatureCostQueryRow {
	dimensions := api.FeatureCostQueryRow_Dimensions{
		Subject:    row.Subject,
		CustomerId: row.CustomerID,
	}

	if len(row.GroupBy) > 0 {
		dimensions.AdditionalProperties = make(map[string]string, len(row.GroupBy))

		for key, value := range row.GroupBy {
			switch key {
			case query.DimensionSubject:
				dimensions.Subject = value
			case query.DimensionCustomerID:
				dimensions.CustomerId = value
			default:
				if value != nil {
					dimensions.AdditionalProperties[key] = *value
				}
			}
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
