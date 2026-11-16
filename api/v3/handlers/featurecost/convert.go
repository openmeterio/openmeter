package featurecost

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/cost"
)

// ToAPIFeatureCostQueryResult converts a domain CostQueryResult to the v3 API response.
func ToAPIFeatureCostQueryResult(result *cost.CostQueryResult, body apiv3models.MeterQueryRequest) apiv3models.FeatureCostQueryResult {
	if result == nil {
		return apiv3models.FeatureCostQueryResult{
			From: body.From,
			To:   body.To,
			Data: []apiv3models.FeatureCostQueryRow{},
		}
	}

	return apiv3models.FeatureCostQueryResult{
		From: body.From,
		To:   body.To,
		Data: lo.Map(result.Rows, func(row cost.CostQueryRow, _ int) apiv3models.FeatureCostQueryRow {
			return toAPIFeatureCostQueryRow(row)
		}),
	}
}

func toAPIFeatureCostQueryRow(row cost.CostQueryRow) apiv3models.FeatureCostQueryRow {
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

	apiRow := apiv3models.FeatureCostQueryRow{
		Usage:      row.Usage.String(),
		Currency:   apiv3models.CurrencyCode(row.Currency),
		From:       row.WindowStart,
		To:         row.WindowEnd,
		Dimensions: dimensions,
	}

	if row.Cost != nil {
		apiRow.Cost = nullable.NewNullableWithValue(row.Cost.String())
	} else {
		apiRow.Cost = nullable.NewNullNullable[apiv3models.Numeric]()
	}

	if row.Detail != "" {
		apiRow.Detail = lo.ToPtr(row.Detail)
	}

	return apiRow
}
